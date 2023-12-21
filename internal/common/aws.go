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

// NewAWSConfig creates and returns a new AWS configuration with default settings.
// It sets the default region and specifies the maximum number of retry attempts for AWS clients.
func NewAWSConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(
		context.TODO(),
		config.WithDefaultRegion(defaultAWSRegion),
		config.WithRetryMaxAttempts(defaultAWSClientRetries),
	)
}
