package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *FalconContainerReconciler) reconcileService(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.Service, error) {
	service := r.newService(falconContainer)
	updated := false
	existingService := &corev1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: injectorName, Namespace: r.Namespace()}, existingService)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, service, r.Scheme); err != nil {
				return &corev1.Service{}, fmt.Errorf("unable to set controller reference on service %s: %v", service.ObjectMeta.Name, err)
			}
			return service, r.Create(ctx, falconContainer, service)
		}
		return &corev1.Service{}, fmt.Errorf("unable to query existing service %s: %v", injectorName, err)
	}
	if !reflect.DeepEqual(service.Spec.Selector, existingService.Spec.Selector) {
		existingService.Spec.Selector = service.Spec.Selector
		updated = true
	}
	if !reflect.DeepEqual(service.Spec.Selector, existingService.Spec.Selector) {
		existingService.Spec.Selector = service.Spec.Selector
		updated = true
	}
	if updated {
		return existingService, r.Update(ctx, falconContainer, existingService)
	}
	return existingService, nil

}

func (r *FalconContainerReconciler) newService(falconContainer *v1alpha1.FalconContainer) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      injectorName,
			Namespace: r.Namespace(),
			Labels:    FcLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: FcLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       common.FalconServiceHTTPSName,
					Port:       *falconContainer.Spec.Injector.ListenPort,
					TargetPort: intstr.FromString(common.FalconServiceHTTPSName),
				},
			},
		},
	}
}
