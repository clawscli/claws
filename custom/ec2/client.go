package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an EC2 client configured for the current context
func GetClient(ctx context.Context) (*ec2.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}
