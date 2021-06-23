package falcon_container_deployer

import (
	"fmt"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/push_auth"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *FalconContainerDeployer) EnsureDockercfg() (push_auth.Credentials, error) {
	namespace := r.Namespace()
	secrets := &corev1.SecretList{}
	err := r.Client.List(r.Ctx, secrets, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	creds := push_auth.GetCredentials(secrets.Items)
	if creds == nil {
		return nil, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
	}
	return creds, nil
}
