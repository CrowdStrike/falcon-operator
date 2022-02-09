package pulltoken

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/go-logr/logr"
)

func MergeAll(ctx context.Context, registryType falconv1alpha1.RegistryTypeSpec, log logr.Logger, query k8s_utils.KubeQuerySecretsMethod) ([]byte, error) {
	secrets, err := query(ctx)
	if err != nil {
		return nil, err
	}
	creds := auth.GetPullCredentials(secrets.Items)

	return auth.MergeCredentials(creds, log)
}
