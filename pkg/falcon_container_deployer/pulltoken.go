package falcon_container_deployer

import (
	"fmt"
	"io/ioutil"
	"os"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/push_auth"
)

const (
	saCertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

func (d *FalconContainerDeployer) pulltoken() (string, error) {
	namespace := d.Namespace()
	secrets := &corev1.SecretList{}
	err := d.Client.List(d.Ctx, secrets, client.InNamespace(namespace))
	if err != nil {
		return "", err
	}
	creds := push_auth.GetCredentials(secrets.Items)
	if creds == nil {
		return "", fmt.Errorf("Cannot find suitable secret in namespace %s to allow falcon-container to pull images from the registry", namespace)
	}
	return creds.Pulltoken()
}

func (d *FalconContainerDeployer) registryCert() ([]byte, error) {
	if _, err := os.Stat(saCertPath); os.IsNotExist(err) {
		return []byte{}, nil
	}
	return ioutil.ReadFile(saCertPath)
}
