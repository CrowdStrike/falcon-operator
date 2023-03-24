package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PullSecret(namespace string, pulltoken []byte) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.FalconPullSecretName,
			Namespace: namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: "secret",
				common.FalconInstanceKey:     common.FalconPullSecretName,
				common.FalconManagedByKey:    common.FalconManagedByValue,
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       common.FalconPartOfValue,
				common.FalconCreatedKey:      common.FalconCreatedValue,
			},
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: common.CleanDecodedBase64(pulltoken),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

}
