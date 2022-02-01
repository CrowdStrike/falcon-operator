package pulltoken

import (
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
)

// CrowdStrike function returns kubernetes pull token for accessing CrowdStrike Falcon Registry.
// Return value is in a form of corev1.SecretTypeDockerConfigJson (.dockerconfigjson)
func CrowdStrike(apiConfig *falcon.ApiConfig) ([]byte, error) {
	registry, err := falcon_registry.NewFalconRegistry(apiConfig)
	if err != nil {
		return nil, err
	}
	return registry.Pulltoken()
}
