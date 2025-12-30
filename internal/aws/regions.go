package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appconfig "github.com/clawscli/claws/internal/config"
)

// CommonRegions is a fallback list of common AWS regions
var CommonRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"eu-west-1",
	"eu-west-2",
	"eu-central-1",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-south-1",
	"sa-east-1",
}

// FetchAvailableRegions fetches available regions from AWS using the current profile.
// Falls back to CommonRegions on error.
func FetchAvailableRegions(ctx context.Context) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(appconfig.Global().Selection())...)
	if err != nil {
		return CommonRegions, nil // Fallback to common regions
	}

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return CommonRegions, nil // Fallback to common regions
	}

	regions := make([]string, 0, len(output.Regions))
	for _, r := range output.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}
	return regions, nil
}
