package container

import (
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerDockerSecrets(name string, namespace string, dockerConfigEntry []byte, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: name,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconManagedByKey:    name,
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

func ContainerTLSSecret(name string, namespace string, falconContainer *falconv1alpha1.FalconContainer) *corev1.Secret {
	altDNS := common.AltDNSListGenerator(falconContainer.Name, namespace)
	fqdn := fmt.Sprintf("%s.%s.svc", falconContainer.Name, namespace)
	ca, certs := common.GenCerts(fmt.Sprintf("%s ca", namespace), fqdn, nil, altDNS, int(falconContainer.Spec.FalconContainerSensorConfig.CertExpiration))
	common.CertAuth = ca

	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels: map[string]string{
			common.FalconInstanceNameKey: name,
			common.FalconInstanceKey:     "container_sensor",
			common.FalconComponentKey:    "container_sensor",
			common.FalconManagedByKey:    name,
			common.FalconProviderKey:     "CrowdStrike",
			common.FalconPartOfKey:       "Falcon",
			common.FalconControllerKey:   "controller-manager",
		},
	},
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte(certs.Cert),
			corev1.TLSPrivateKeyKey: []byte(certs.Key),
		},
		Type: corev1.SecretTypeOpaque,
	}
}
