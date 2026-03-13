package pushtoken

import (
	"context"
	"fmt"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
)

// GetCredentials returns container pushtoken authentication information that can be used to authenticate container push requests.
func GetCredentials(ctx context.Context, registryType falconv1alpha1.RegistryTypeSpec, query k8s_utils.KubeQuerySecretsMethod) (auth.Credentials, error) {
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

		secretCount := len(secrets.Items)

		var secretNames []string
		for _, secret := range secrets.Items {
			secretNames = append(secretNames, secret.Name)
		}

		creds := auth.GetPushCredentials(secrets.Items)
		if creds == nil {
			return nil, fmt.Errorf("Cannot find suitable secret to push falcon-image to your registry (found %d secrets: %s)", secretCount, strings.Join(secretNames, ", "))
		}
		return creds, nil
	}
}
