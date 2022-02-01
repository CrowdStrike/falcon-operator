package pulltoken

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
)

func MergeAll(ctx context.Context, registryType falconv1alpha1.RegistryTypeSpec, apiConfig *falcon.ApiConfig, log logr.Logger, query k8s_utils.KubeQuerySecretsMethod) ([]byte, error) {
	secrets, err := query(ctx)
	if err != nil {
		return nil, err
	}
	pullTokens := [][]byte{}
	for _, cred := range auth.GetPullCredentials(secrets.Items) {
		log.Info("Found pull secret to be forwarded to Falcon Container Injector: ", "secret.Name", cred.Name())
		pullToken, err := cred.Pulltoken()
		if err != nil {
			log.Error(err, "Skipping pull secret", "secret.Name", cred.Name())
		} else {
			pullTokens = append(pullTokens, pullToken)
		}
	}
	if registryType == falconv1alpha1.RegistryTypeCrowdStrike {
		pullToken, err := CrowdStrike(apiConfig)
		if err != nil {
			return nil, err
		}
		pullTokens = append(pullTokens, pullToken)
	}

	return auth.MergePullTokens(pullTokens)
}
