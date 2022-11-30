package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	injectorNamespace = "falcon-system"
)

var (
	namespaceLabels = map[string]string{
		"sensor.falcon-system.crowdstrike.com/injection": "disabled",
		"kubernetes.io/metadata.name":                    "falcon-system",
	}
)

func (r *FalconContainerReconciler) Namespace() string {
	return injectorNamespace
}

func (r *FalconContainerReconciler) NamespaceLabels() map[string]string {
	nsLabels := make(map[string]string)
	for k, v := range FcLabels {
		nsLabels[k] = v
	}
	for k, v := range namespaceLabels {
		nsLabels[k] = v
	}
	return nsLabels
}

func (r *FalconContainerReconciler) reconcileNamespace(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.Namespace, error) {
	namespace := r.newNamespace()
	existingNamespace := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: r.Namespace()}, existingNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, namespace, r.Scheme); err != nil {
				return &corev1.Namespace{}, fmt.Errorf("unable to set controller reference on namespace %s: %v", namespace.ObjectMeta.Name, err)
			}
			return namespace, r.Create(ctx, falconContainer, namespace)
		}
		return &corev1.Namespace{}, fmt.Errorf("unable to query existing namespace %s: %v", r.Namespace(), err)
	}
	if !reflect.DeepEqual(namespace.ObjectMeta.Labels, existingNamespace.ObjectMeta.Labels) {
		existingNamespace.ObjectMeta.Labels = namespace.ObjectMeta.Labels
		return existingNamespace, r.Update(ctx, falconContainer, existingNamespace)
	}
	return existingNamespace, nil
}

func (r *FalconContainerReconciler) newNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   r.Namespace(),
			Labels: r.NamespaceLabels(),
		},
	}
}
