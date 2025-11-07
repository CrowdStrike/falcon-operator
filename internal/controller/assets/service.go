package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Service returns a Kubernetes Service object
func Service(name string, namespace string, component string, selector map[string]string, portName string, port int32) *corev1.Service {
	labels := common.CRLabels("service", name, component)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       portName,
					Port:       port,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(common.FalconServiceHTTPSName),
				},
			},
		},
	}
}

// ServiceWithCustomLabels returns a Kubernetes Service object with custom labels
func ServiceWithCustomLabels(
	name string,
	namespace string,
	selector map[string]string,
	labels map[string]string,
	servicePortName string,
	targetPortName string,
	port int32,
) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       servicePortName,
					Port:       port,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(targetPortName),
				},
			},
		},
	}
}
