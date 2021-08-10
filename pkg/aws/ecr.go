package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
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
