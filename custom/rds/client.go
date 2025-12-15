package rds

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an RDS client configured for the current context
func GetClient(ctx context.Context) (*rds.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return rds.NewFromConfig(cfg), nil
}
