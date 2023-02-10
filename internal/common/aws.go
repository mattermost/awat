package common

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	defaultAWSRegion        = "us-east-1"
	defaultAWSClientRetries = 3
)

func NewAWSConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(
		context.TODO(),
		config.WithDefaultRegion(defaultAWSRegion),
		config.WithRetryMaxAttempts(defaultAWSClientRetries),
	)
}
