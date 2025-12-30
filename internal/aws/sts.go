package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// FetchAccountID fetches the AWS account ID using STS GetCallerIdentity.
// Returns empty string on error.
func FetchAccountID(ctx context.Context, cfg aws.Config) string {
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil || identity.Account == nil {
		return ""
	}
	return *identity.Account
}

func FetchAccountIDForContext(ctx context.Context) string {
	cfg, err := NewConfig(ctx)
	if err != nil {
		return ""
	}
	return FetchAccountID(ctx, cfg)
}
