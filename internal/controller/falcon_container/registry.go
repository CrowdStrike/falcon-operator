package falcon

import (
	"context"
	"fmt"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *FalconContainerReconciler) reconcileRegistrySecrets(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*corev1.SecretList, error) {
	injectionEnabledValue := "enabled"
	injectionDisabledValue := "disabled"
	disableDefaultNSInjection := false
	secretList := &corev1.SecretList{}

	if falconContainer.Spec.Injector.DisableDefaultNSInjection {
		disableDefaultNSInjection = falconContainer.Spec.Injector.DisableDefaultNSInjection
	}

	nsList := &corev1.NamespaceList{}
	if err := r.Reader.List(ctx, nsList); err != nil {
		return &corev1.SecretList{}, fmt.Errorf("unable to list current namespaces: %v", err)
	}

	falconApiConfig, apiConfigErr := r.falconApiConfig(ctx, falconContainer)
	if apiConfigErr != nil {
		return &corev1.SecretList{}, apiConfigErr
	}

	pulltoken, err := pulltoken.CrowdStrike(ctx, falconApiConfig)
	if err != nil {
		return &corev1.SecretList{}, fmt.Errorf("unable to get registry pull token: %v", err)
	}

	for _, ns := range nsList.Items {
		if ns.Name == "kube-public" || ns.Name == "kube-system" {
			continue
		}
		if disableDefaultNSInjection {
			// if default namespace injection is disabled, require that the injection label be set to enabled
			// in both cases below, ensure that we're not blocking pull secret creation within the injector namespace
			if (ns.Labels == nil || ns.Labels[common.FalconContainerInjection] != injectionEnabledValue) && ns.Name != falconContainer.Spec.InstallNamespace {
				continue
			}
		} else {
			// otherwise, just ensure the injection label is not set to disabled
			if ns.Labels != nil && ns.Labels[common.FalconContainerInjection] == injectionDisabledValue && ns.Name != falconContainer.Spec.InstallNamespace {
				continue
			}
		}

		secret, err := r.reconcileRegistrySecret(ns.Name, pulltoken, ctx, log, falconContainer)
		if err != nil {
			return secretList, fmt.Errorf("unable to reconcile registry secret in namespace %s: %v", ns.Name, err)
		}

		secretList.Items = append(secretList.Items, *secret)
	}

	return secretList, nil
}

func (r *FalconContainerReconciler) reconcileRegistrySecret(namespace string, pulltoken []byte, ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*corev1.Secret, error) {
	secretData := map[string][]byte{corev1.DockerConfigJsonKey: common.CleanDecodedBase64(pulltoken)}
	secret := assets.Secret(common.FalconPullSecretName, namespace, "falcon-operator", secretData, corev1.SecretTypeDockerConfigJson)
	existingSecret := &corev1.Secret{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.FalconPullSecretName, Namespace: namespace}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(falconContainer, secret, r.Scheme); err != nil {
				return &corev1.Secret{}, fmt.Errorf("failed to set controller reference on registry pull token secret %s: %v", secret.ObjectMeta.Name, err)
			}

			return secret, r.Create(ctx, log, falconContainer, secret)
		}

		return &corev1.Secret{}, fmt.Errorf("unable to query existing secret %s in namespace %s: %v", common.FalconPullSecretName, namespace, err)
	}

	if reflect.DeepEqual(secret.Data, existingSecret.Data) {
		return existingSecret, nil
	}

	existingSecret.Data = secret.Data

	return existingSecret, r.Update(ctx, log, falconContainer, existingSecret)
}
