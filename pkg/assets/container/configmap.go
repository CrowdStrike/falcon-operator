package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerConfigMap(dsname string, nsname string, falconContainer *falconv1alpha1.FalconContainer) *corev1.ConfigMap {
	return containerConfigMap(dsname, nsname, falconContainer)
}

func containerConfigMap(dsname string, nsname string, falconContainer *falconv1alpha1.FalconContainer) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsname,
			Namespace: nsname,
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsname,
				common.FalconInstanceKey:     "kernel_sensor",
				common.FalconComponentKey:    "kernel_sensor",
				common.FalconManagedByKey:    dsname,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Data: map[string]string{
			"CP_NAMESPACE":                nsname,
			"FALCON_INJECTOR_LISTEN_PORT": "4433",
		},
	}
}
