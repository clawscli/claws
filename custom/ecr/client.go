package ecr

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecr"

	appaws "github.com/clawscli/claws/internal/aws"
)

func GetClient(ctx context.Context) (*ecr.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return ecr.NewFromConfig(cfg), nil
}
