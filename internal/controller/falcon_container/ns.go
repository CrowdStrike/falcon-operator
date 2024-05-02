package falcon

import (
	"context"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	namespaceLabels = map[string]string{
		common.FalconContainerInjection: "disabled",
		"kubernetes.io/metadata.name":   "falcon-system",
	}
)

func (r *FalconContainerReconciler) NamespaceLabels(falconContainer *falconv1alpha1.FalconContainer) map[string]string {
	nsLabels := common.CRLabels("namespace", falconContainer.Spec.InstallNamespace, common.FalconSidecarSensor)
	for k, v := range namespaceLabels {
		nsLabels[k] = v
	}
	return nsLabels
}

func (r *FalconContainerReconciler) reconcileNamespace(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*corev1.Namespace, error) {
	namespace := r.newNamespace(falconContainer)
	existingNamespace := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: falconContainer.Spec.InstallNamespace}, existingNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, namespace, r.Scheme); err != nil {
				return &corev1.Namespace{}, fmt.Errorf("unable to set controller reference on namespace %s: %v", namespace.ObjectMeta.Name, err)
			}
			return namespace, r.Create(ctx, log, falconContainer, namespace)
		}
		return &corev1.Namespace{}, fmt.Errorf("unable to query existing namespace %s: %v", falconContainer.Spec.InstallNamespace, err)
	}

	return existingNamespace, nil
}

func (r *FalconContainerReconciler) newNamespace(falconContainer *falconv1alpha1.FalconContainer) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   falconContainer.Spec.InstallNamespace,
			Labels: r.NamespaceLabels(falconContainer),
		},
	}
}
