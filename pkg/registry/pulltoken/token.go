package pulltoken

import (
	"context"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/gofalcon/falcon"
)

func Get(ctx context.Context, registryType falconv1alpha1.RegistryTypeSpec, apiConfig *falcon.ApiConfig, query k8s_utils.KubeQuerySecretsMethod) ([]byte, error) {
	switch registryType {
	case falconv1alpha1.RegistryTypeECR:
		return nil, nil
	case falconv1alpha1.RegistryTypeCrowdStrike:
		return CrowdStrike(apiConfig)
	default:
		secrets, err := query(ctx)
		if err != nil {
			return nil, err
		}
		creds := auth.GetPushCredentials(secrets.Items)
		if creds == nil {
			return nil, fmt.Errorf("Cannot find suitable secret to allow falcon-container to pull images from the registry")
		}
		return creds.Pulltoken()
	}
}
