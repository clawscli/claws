package elbv2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an ELBv2 client configured for the current context
func GetClient(ctx context.Context) (*elasticloadbalancingv2.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return elasticloadbalancingv2.NewFromConfig(cfg), nil
}
