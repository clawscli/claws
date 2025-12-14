package ecs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an ECS client configured for the current context
func GetClient(ctx context.Context) (*ecs.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return ecs.NewFromConfig(cfg), nil
}
