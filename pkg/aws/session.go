package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Config struct {
	aws.Config
}

func NewConfig() (*Config, error) {
	if os.Getenv("AWS_REGION") == "" {
		return nil, fmt.Errorf("Environment variable AWS_REGION is not set for the operator. This is indicator of misconfiguration. Please ensure kubernetes service account bound to the operator is configured by eksctl create iamserviceaccount as described in the documentation.")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return &Config{
		Config: cfg,
	}, nil

}
