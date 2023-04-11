package falcon

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
)

func (r *FalconContainerReconciler) UpsertECRRepo(ctx context.Context) (*types.Repository, error) {
	cfg, err := aws.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialise connection to AWS. Please make sure that kubernetes service account falcon-operator has access to AWS IAM role and OIDC Identity provider is running on the cluster. Error was: %v", err)
	}

	data, err := cfg.UpsertRepository(ctx, "falcon-container")
	if err != nil {
		return nil, fmt.Errorf("Failed to upsert ECR repository: %v", err)
	}

	return data, nil
}
