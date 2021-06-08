package falcon_container_deployer

import (
	"context"
	"fmt"
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *FalconContainerDeployer) EnsureDockercfg(ctx context.Context, namespace string) error {
	dockercfg, err := r.getDockercfg(ctx, namespace)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/tmp/.dockercfg", dockercfg, 0600)
}

func (r *FalconContainerDeployer) getDockercfg(ctx context.Context, namespace string) ([]byte, error) {
	secrets := &corev1.SecretList{}
	err := r.Client.List(ctx, secrets, client.InNamespace(namespace))
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
