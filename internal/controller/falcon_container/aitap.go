package falcon

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var aitapExcludedNamespaces = []string{
	"kube-public",
	"kube-system",
	"kube-node-lease",
	"falcon-system",
	"falcon-kac",
	"falcon-iar",
}

func validateAITapConfig(aitap falconv1alpha1.AITapSpec) error {
	if aitap.Namespaces != "" && aitap.AllNamespaces {
		return fmt.Errorf("AITap: 'namespaces' and 'allNamespaces' cannot both be set")
	}
	if aitap.Namespaces != "" || aitap.AllNamespaces {
		if aitap.AidrCollectorBaseApiUrl == "" {
			return fmt.Errorf("AITap: 'aidrCollectorBaseApiUrl' is required when 'namespaces' or 'allNamespaces' is set")
		}
		if !aitap.UseExistingSecret && aitap.AidrCollectorApiToken == "" {
			return fmt.Errorf("AITap: 'aidrCollectorApiToken' is required when 'namespaces' or 'allNamespaces' is set and 'useExistingSecret' is false")
		}
	}
	if aitap.UseExistingSecret && aitap.AidrSecretName == "" {
		return fmt.Errorf("AITap: 'aidrSecretName' is required when 'useExistingSecret' is true")
	}
	return nil
}

func (r *FalconContainerReconciler) reconcileAITapSecrets(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*corev1.SecretList, error) {
	secretList := &corev1.SecretList{}

	if err := validateAITapConfig(falconContainer.Spec.Injector.AITap); err != nil {
		return secretList, err
	}

	if falconContainer.Spec.Injector.AITap.UseExistingSecret {
		log.Info("AITap configured to use existing secret, skipping secret creation")
		return secretList, nil
	}

	if falconContainer.Spec.Injector.AITap.AidrCollectorApiToken == "" {
		log.Info("AITap AI-DR token not configured, skipping AITap secret reconciliation")
		return secretList, nil
	}

	targetNamespaces, err := r.getAITapTargetNamespaces(ctx, log, falconContainer)
	if err != nil {
		return secretList, fmt.Errorf("unable to determine target namespaces for AITap secrets: %v", err)
	}

	secretName := r.getAITapSecretName(falconContainer)

	for _, ns := range targetNamespaces {
		secret, err := r.reconcileAITapSecret(ctx, log, falconContainer, ns, secretName)
		if err != nil {
			return secretList, fmt.Errorf("unable to reconcile AITap secret in namespace %s: %v", ns, err)
		}
		secretList.Items = append(secretList.Items, *secret)
	}

	log.Info("Successfully reconciled AITap AI-DR secrets", "namespaces", len(targetNamespaces), "secretName", secretName)
	return secretList, nil
}

func (r *FalconContainerReconciler) getAITapTargetNamespaces(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) ([]string, error) {
	excludedNamespaces := append(aitapExcludedNamespaces, falconContainer.Spec.InstallNamespace)

	if falconContainer.Spec.Injector.AITap.AllNamespaces {
		nsList := &corev1.NamespaceList{}
		if err := r.Reader.List(ctx, nsList); err != nil {
			return nil, fmt.Errorf("unable to list namespaces: %v", err)
		}
		var targetNamespaces []string
		for _, ns := range nsList.Items {
			if slices.Contains(excludedNamespaces, ns.Name) || strings.HasPrefix(ns.Name, "openshift") {
				continue
			}

			targetNamespaces = append(targetNamespaces, ns.Name)
		}
		log.Info("AITap AllNamespaces enabled", "count", len(targetNamespaces))
		return targetNamespaces, nil
	}

	if falconContainer.Spec.Injector.AITap.Namespaces != "" {
		var targetNamespaces []string
		for _, ns := range strings.Split(falconContainer.Spec.Injector.AITap.Namespaces, ",") {
			ns = strings.TrimSpace(ns)
			if ns == "" || slices.Contains(excludedNamespaces, ns) || strings.HasPrefix(ns, "openshift") {
				continue
			}

			targetNamespaces = append(targetNamespaces, ns)
		}
		log.Info("AITap namespaces configured", "namespaces", targetNamespaces)
		return targetNamespaces, nil
	}

	return nil, nil
}

func (r *FalconContainerReconciler) getAITapSecretName(falconContainer *falconv1alpha1.FalconContainer) string {
	if falconContainer.Spec.Injector.AITap.AidrSecretName != "" {
		return falconContainer.Spec.Injector.AITap.AidrSecretName
	}

	if falconContainer.Spec.Injector.GKEAutopilot {
		return common.GKEAutoPilotAITapAidrSecretName
	}

	return common.FalconAITapAidrSecretName
}

func (r *FalconContainerReconciler) reconcileAITapSecret(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer, namespace string, secretName string) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		".collector-aidr-token": []byte(falconContainer.Spec.Injector.AITap.AidrCollectorApiToken),
	}

	secret := assets.Secret(secretName, namespace, common.FalconSidecarSensor, secretData, corev1.SecretTypeOpaque)
	existingSecret := &corev1.Secret{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: secretName, Namespace: namespace}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(falconContainer, secret, r.Scheme); err != nil {
				return &corev1.Secret{}, fmt.Errorf("failed to set controller reference on AITap secret %s: %v", secret.ObjectMeta.Name, err)
			}
			log.Info("Creating AITap AI-DR secret", "namespace", namespace, "secretName", secretName)
			return secret, r.Create(ctx, log, falconContainer, secret)
		}
		return &corev1.Secret{}, fmt.Errorf("unable to query existing AITap secret %s in namespace %s: %v", secretName, namespace, err)
	}

	if !reflect.DeepEqual(secret.Data, existingSecret.Data) {
		existingSecret.Data = secret.Data
		log.Info("Updating AITap AI-DR secret", "namespace", namespace, "secretName", secretName)
		return existingSecret, r.Update(ctx, log, falconContainer, existingSecret)
	}

	return existingSecret, nil
}
