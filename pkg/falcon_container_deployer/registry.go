package falcon_container_deployer

import (
	"encoding/base64"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
)

const (
	saCertDir  = "/var/run/secrets/kubernetes.io/serviceaccount/"
	saCertPath = saCertDir + "ca.crt"
)

func (d *FalconContainerDeployer) pulltoken() ([]byte, error) {
	switch d.Instance.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeECR:
		return nil, nil
	case falconv1alpha1.RegistryTypeCrowdStrike:
		return pulltoken.CrowdStrike(d.falconApiConfig())
	default:
		namespace := d.Namespace()
		secrets := &corev1.SecretList{}
		err := d.Client.List(d.Ctx, secrets, client.InNamespace(namespace))
		if err != nil {
			return nil, err
		}
		creds := auth.GetPushCredentials(secrets.Items)
		if creds == nil {
			return nil, fmt.Errorf("Cannot find suitable secret in namespace %s to allow falcon-container to pull images from the registry", namespace)
		}
		return creds.Pulltoken()
	}
}

func (d *FalconContainerDeployer) pulltokenBase64() (string, error) {
	token, err := d.pulltoken()
	if err != nil || token == nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(token), nil
}

func (d *FalconContainerDeployer) registryCertExists() bool {
	_, err := os.Stat(saCertPath)
	if err != nil && !os.IsNotExist(err) {
		d.Log.Error(err, "Received error when trying to stat k8s certificate", "path", saCertPath)
	}
	return err == nil
}
