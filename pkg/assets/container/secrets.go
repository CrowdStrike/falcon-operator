package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerDockerSecrets(dsName string, nsName string, dockerConfigEntry []byte, falconsensor *falconv1alpha1.FalconSensor) *corev1.Secret {
	return containerDockerSecrets(dsName, nsName, dockerConfigEntry, falconsensor)
}

func containerDockerSecrets(dsName string, nsName string, dockerConfigEntry []byte, falconsensor *falconv1alpha1.FalconSensor) *corev1.Secret {
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
