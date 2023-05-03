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
			Labels:    common.CRLabels("configmap", dsname, common.FalconKernelSensor),
		},
		Data: config.SensorEnvVars(),
	}
}
