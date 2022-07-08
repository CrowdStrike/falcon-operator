package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerConfigMap(name string, falconContainer *falconv1alpha1.FalconContainer) *corev1.ConfigMap {
	configData := common.FalconContainerConfig(falconContainer)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: falconContainer.Namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: name,
				common.FalconInstanceKey:     "kernel_sensor",
				common.FalconComponentKey:    "kernel_sensor",
				common.FalconManagedByKey:    name,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Data: configData,
	}
}
