package falcon_container_deployer

import (
	"fmt"
	"io/ioutil"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/push_auth"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *FalconContainerDeployer) EnsureDockercfg() (push_auth.Credentials, error) {
	dockercfg, err := r.getDockercfg()
	if err != nil {
		return nil, err
	}
	return &push_auth.Legacy{}, ioutil.WriteFile("/tmp/.dockercfg", dockercfg, 0600)
}

func (r *FalconContainerDeployer) getDockercfg() ([]byte, error) {
	namespace := r.Namespace()
	secrets := &corev1.SecretList{}
	err := r.Client.List(r.Ctx, secrets, client.InNamespace(namespace))
	if err != nil {
		return []byte{}, err
	}

	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}
		if secret.Type != "kubernetes.io/dockercfg" && secret.Type != "kubernetes.io/dockerconfigjson" {
			continue
		}

		if (secret.ObjectMeta.Annotations == nil || secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"] != "builder") && secret.Name != "builder" {
			continue
		}

		value, ok := secret.Data[".dockercfg"]
		if ok {
			return value, nil
		}
		value, ok = secret.Data[".dockerconfigjson"]
		if ok {
			return value, nil
		}
	}

	return []byte{}, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
}
