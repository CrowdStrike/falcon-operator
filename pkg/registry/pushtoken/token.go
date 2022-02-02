package pushtoken

import (
	"context"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
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

		creds := auth.GetPushCredentials(secrets.Items)
		if creds == nil {
			return nil, fmt.Errorf("Cannot find suitable secret to push falcon-image to your registry")
		}
		return creds, nil
	}
}
