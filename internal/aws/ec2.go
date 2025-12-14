package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2TagValue returns the value of an EC2 tag with the given key.
// Returns empty string if the tag is not found.
func EC2TagValue(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == key {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

// EC2NameTag returns the value of the "Name" tag for EC2 resources.
func EC2NameTag(tags []types.Tag) string {
	return EC2TagValue(tags, "Name")
}
