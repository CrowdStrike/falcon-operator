package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Secret returns a Kubernetes Secret object
func Secret(name string, namespace string, component string, data map[string][]byte, sType corev1.SecretType) *corev1.Secret {
	labels := common.CRLabels("secret", name, component)

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
		Type: sType,
	}
}
