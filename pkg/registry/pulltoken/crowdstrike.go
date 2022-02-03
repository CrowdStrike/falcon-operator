package pulltoken

import (
	"context"

	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
)

// CrowdStrike function returns kubernetes pull token for accessing CrowdStrike Falcon Registry.
// Return value is in a form of corev1.SecretTypeDockerConfigJson (.dockerconfigjson)
func CrowdStrike(ctx context.Context, apiConfig *falcon.ApiConfig) ([]byte, error) {
	apiConfig.Context = ctx
	registry, err := falcon_registry.NewFalconRegistry(apiConfig)
	if err != nil {
		return nil, err
	}
	return registry.Pulltoken()
}
