package pushtoken

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
)

type KubeQuerySecretsMethod func(ctx context.Context) (*corev1.SecretList, error)

func QuerySecrets(namespace string, cli client.Client) KubeQuerySecretsMethod {
	return func(ctx context.Context) (*corev1.SecretList, error) {
		secrets := &corev1.SecretList{}
		err := cli.List(ctx, secrets, client.InNamespace(namespace))
		if err != nil {
			return nil, err
		}
		return secrets, nil
	}
}

// GetCredentials returns container pushtoken authentication information that can be used to authenticate container push requests.
func GetCredentials(ctx context.Context, registryType falconv1alpha1.RegistryTypeSpec, query KubeQuerySecretsMethod) (auth.Credentials, error) {
	switch registryType {
	case falconv1alpha1.RegistryTypeECR:
		cfg, err := aws.NewConfig()
		if err != nil {
			return nil, err
		}
		token, err := cfg.ECRLogin(ctx)
		if err != nil {
			return nil, err
		}
		return auth.ECRCredentials(string(token))
	default:
		secrets, err := query(ctx)
		if err != nil {
			return nil, err
		}

		creds := auth.GetCredentials(secrets.Items)
		if creds == nil {
			return nil, fmt.Errorf("Cannot find suitable secret to push falcon-image to your registry")
		}
		return creds, nil
	}
}
