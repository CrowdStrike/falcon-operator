package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ContainerService(dsName string, nsName string, falconsensor *falconv1alpha1.FalconSensor) *corev1.Service {
	return containerService(dsName, nsName, falconsensor)
}

func containerService(dsName string, nsName string, falconsensor *falconv1alpha1.FalconSensor) *corev1.Service {
	portName := "https"
	var port int32 = 443

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: nsName,
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsName,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconManagedByKey:    dsName,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				common.FalconInstanceNameKey: dsName,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       portName,
					Port:       port,
					TargetPort: intstr.FromString(portName),
				},
			},
		},
	}
}
