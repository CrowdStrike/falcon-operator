package falcon_container_deployer

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/registry_auth"
)

const (
	saCertDir  = "/var/run/secrets/kubernetes.io/serviceaccount/"
	saCertPath = saCertDir + "ca.crt"
)

func (d *FalconContainerDeployer) pulltokenBase64() (string, error) {
	switch d.Instance.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeECR:
		return "", nil
	default:
		namespace := d.Namespace()
		secrets := &corev1.SecretList{}
		err := d.Client.List(d.Ctx, secrets, client.InNamespace(namespace))
		if err != nil {
			return "", err
		}
		creds := registry_auth.GetCredentials(secrets.Items)
		if creds == nil {
			return "", fmt.Errorf("Cannot find suitable secret in namespace %s to allow falcon-container to pull images from the registry", namespace)
		}
		return creds.Pulltoken()
	}
}

func (d *FalconContainerDeployer) registryCertExists() bool {
	_, err := os.Stat(saCertPath)
	if err != nil && !os.IsNotExist(err) {
		d.Log.Error(err, "Received error when trying to stat k8s certificate", "path", saCertPath)
	}
	return err == nil
}
