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
	creds := auth.GetPullCredentials(secrets.Items)
	/*
		if registryType == falconv1alpha1.RegistryTypeCrowdStrike {
			crwd, err := crowdStrikeCreds(apiConfig)
			if err != nil {
				return nil, err
			}
			creds = append(creds, crwd)
		}
	*/

	return auth.MergeCredentials(creds, log)
}
