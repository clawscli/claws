package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns a DynamoDB client configured for the current context
func GetClient(ctx context.Context) (*dynamodb.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return dynamodb.NewFromConfig(cfg), nil
}
