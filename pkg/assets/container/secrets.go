package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerDockerSecrets(dsName string, nsName string, dockerConfigEntry []byte, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	return &corev1.Secret{
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
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigEntry,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
}

func ContainerTLSSecret(dsName string, nsName string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	cert := common.GenCert(nsName, nil, nil, int(falconContainer.Spec.FalconContainerSensorConfig.CertExpiration), common.CA)
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
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
		Data: map[string][]byte{
			corev1.TLSCertKey:       common.EncodedBase64String(cert.Cert),
			corev1.TLSPrivateKeyKey: common.EncodedBase64String(cert.Key),
		},
		Type: corev1.SecretTypeTLS,
	}
}
