package falcon_container_deployer

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
)

func (d *FalconContainerDeployer) UpsertECRRepo() (*types.Repository, error) {
	cfg, err := aws.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialise connection to AWS. Please make sure that kubernetes service account falcon-operator has access to AWS IAM role and OIDC Identity provider is running on the cluster. Error was: %v", err)
	}
	data, err := cfg.UpsertRepository(d.Ctx, "falcon-container")
	if err != nil {
		return nil, fmt.Errorf("Failed to upsert ECR repository: %v", err)
	}
	return data, nil
}
