package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/node"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DaemonsetConfigMap(dsname string, nsname string, config *node.ConfigCache) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsname,
			Namespace: nsname,
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsname,
				common.FalconInstanceKey:     "kernel_sensor",
				common.FalconComponentKey:    "kernel_sensor",
				common.FalconManagedByKey:    dsname,
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Data: config.SensorEnvVars(),
	}
}
