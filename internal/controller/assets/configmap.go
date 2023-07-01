package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SensorConfigMap returns a ConfigMap object for the sensor configuration
func SensorConfigMap(name string, ns string, component string, data map[string]string) *corev1.ConfigMap {
	labels := common.CRLabels("configmap", name, component)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Data: data,
	}
}
