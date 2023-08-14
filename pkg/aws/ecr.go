package aws

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecr_types "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func (c *Config) UpsertRepository(ctx context.Context, name string) (*ecr_types.Repository, error) {
	client := ecr.NewFromConfig(c.Config)

	describeOutput, err := client.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{name},
	})
	if err == nil && describeOutput != nil && len(describeOutput.Repositories) == 1 {
		return &describeOutput.Repositories[0], nil
	}

	createOutput, err := client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: &name,
	})

	if err != nil {
		return nil, fmt.Errorf("Could not create ECR repository %s: %v", name, err)
	}

	return createOutput.Repository, nil
}

func (c *Config) ECRLogin(ctx context.Context) ([]byte, error) {
	client := ecr.NewFromConfig(c.Config)
	output, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, fmt.Errorf("Cannot fetch authorization token for ECR: %v", err)
	}
	if output == nil || len(output.AuthorizationData) < 1 || output.AuthorizationData[0].AuthorizationToken == nil || len(*output.AuthorizationData[0].AuthorizationToken) == 0 {
		return nil, fmt.Errorf("Cannot get authorization token fro ECR.")
	}
	return base64.StdEncoding.DecodeString(*output.AuthorizationData[0].AuthorizationToken)
}

func UpsertECRRepo(ctx context.Context, name string) (*types.Repository, error) {
	cfg, err := NewConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialise connection to AWS. Please make sure that kubernetes service account falcon-operator has access to AWS IAM role and OIDC Identity provider is running on the cluster. Error was: %v", err)
	}

	data, err := cfg.UpsertRepository(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("Failed to upsert ECR repository: %v", err)
	}

	return data, nil
}
