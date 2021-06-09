package falcon_container_deployer

import (
	"fmt"
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *FalconContainerDeployer) EnsureDockercfg() error {
	dockercfg, err := r.getDockercfg()
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/tmp/.dockercfg", dockercfg, 0600)
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
		if secret.Type != "kubernetes.io/dockercfg" {
			continue
		}

		if secret.ObjectMeta.Annotations == nil || secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"] != "builder" {
			continue
		}

		value, ok := secret.Data[".dockercfg"]
		if !ok {
			continue
		}
		return value, nil
	}

	return []byte{}, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
}
