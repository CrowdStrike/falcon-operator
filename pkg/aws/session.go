package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	//"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	//"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Config struct {
	aws.Config
}

func NewConfig() (*Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return &Config{
		Config: cfg,
	}, nil

}
