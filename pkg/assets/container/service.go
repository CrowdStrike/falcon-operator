package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ContainerService(name string, namespace string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: name,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconManagedByKey:    name,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":                        name,
				common.FalconInstanceNameKey: namespace,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconProviderKey:     "CrowdStrike",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       common.FalconServiceHTTPSName,
					Port:       common.FalconServiceHTTPSPort,
					TargetPort: intstr.FromString(common.FalconServiceHTTPSName),
				},
			},
		},
	}
}
