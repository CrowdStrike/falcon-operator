package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerDockerSecrets(dsName string, nsName string, dockerConfigEntry string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	return containerDockerSecrets(dsName, nsName, dockerConfigEntry, falconContainer)
}

func ContainerTLSSecret(dsName string, nsName string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	return containerTLSSecret(dsName, nsName, falconContainer)
}

func containerDockerSecrets(dsName string, nsName string, dockerConfigEntry string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {

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
			corev1.DockerConfigJsonKey: []byte(dockerConfigEntry),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
}

func containerTLSSecret(dsName string, nsName string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	cert := common.GenCert(nsName, nil, nil, common.Validity, common.CA)
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
			"tls.crt": common.EncodedBase64String(cert.Cert),
			"tls.key": common.EncodedBase64String(cert.Key),
		},
		Type: corev1.SecretTypeOpaque,
	}
}
