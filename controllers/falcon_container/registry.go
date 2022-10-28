package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/assets"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *FalconContainerReconciler) reconcileRegistrySecrets(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.SecretList, error) {
	injectionLabelKey := fmt.Sprintf("sensor.%s.crowdstrike.com/injection", r.Namespace())
	injectionEnabledValue := "enabled"
	injectionDisabledValue := "disabled"
	disableDefaultNSInjection := false
	secretList := &corev1.SecretList{}
	if falconContainer.Spec.Injector.DisableDefaultNSInjection {
		disableDefaultNSInjection = falconContainer.Spec.Injector.DisableDefaultNSInjection
	}
	nsList := &corev1.NamespaceList{}
	if err := r.Client.List(ctx, nsList); err != nil {
		return &corev1.SecretList{}, fmt.Errorf("unable to list current namespaces: %v", err)
	}

	pulltoken, err := pulltoken.CrowdStrike(ctx, r.falconApiConfig(ctx, falconContainer))
	if err != nil {
		return &corev1.SecretList{}, fmt.Errorf("unable to get registry pull token: %v", err)
	}

	for _, ns := range nsList.Items {
		if ns.Name == "kube-public" {
			continue
		}
		if disableDefaultNSInjection {
			// if default namespace injection is disabled, require that the injection label be set to enabled
			// in both cases below, ensure that we're not blocking pull secret creation within the injector namespace
			if (ns.Labels == nil || ns.Labels[injectionLabelKey] != injectionEnabledValue) && ns.Name != r.Namespace() {
				continue
			}
		} else {
			// otherwise, just ensure the injection label is not set to disabled
			if ns.Labels != nil && ns.Labels[injectionLabelKey] == injectionDisabledValue && ns.Name != r.Namespace() {
				continue
			}
		}
		secret, err := r.reconcileRegistrySecret(ns.Name, pulltoken, ctx, falconContainer)
		if err != nil {
			return secretList, fmt.Errorf("unable to reconcile registry secret in namespace %s: %v", ns.Name, err)
		}
		secretList.Items = append(secretList.Items, *secret)
	}
	return secretList, nil
}

func (r *FalconContainerReconciler) reconcileRegistrySecret(namespace string, pulltoken []byte, ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.Secret, error) {
	secret := assets.PullSecret(namespace, pulltoken)
	existingSecret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.FalconPullSecretName, Namespace: namespace}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(falconContainer, &secret, r.Scheme); err != nil {
				return &corev1.Secret{}, fmt.Errorf("failed to set controller reference on registry pull token secret %s: %v", secret.ObjectMeta.Name, err)
			}
			return &secret, r.Create(ctx, falconContainer, &secret)
		}
		return &corev1.Secret{}, fmt.Errorf("unable to query existing secret %s in namespace %s: %v", common.FalconPullSecretName, namespace, err)
	}
	if reflect.DeepEqual(secret.Data, existingSecret.Data) {
		return existingSecret, nil
	}
	existingSecret.Data = secret.Data
	return existingSecret, r.Update(ctx, falconContainer, existingSecret)
}
