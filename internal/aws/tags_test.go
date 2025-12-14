package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestTagsToMap_EC2(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: aws.String("Name"), Value: aws.String("test-instance")},
		{Key: aws.String("Environment"), Value: aws.String("production")},
		{Key: aws.String("Team"), Value: aws.String("platform")},
	}

	result := TagsToMap(tags)

	if len(result) != 3 {
		t.Errorf("TagsToMap() returned %d tags, want 3", len(result))
	}

	expected := map[string]string{
		"Name":        "test-instance",
		"Environment": "production",
		"Team":        "platform",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("TagsToMap()[%q] = %q, want %q", k, result[k], v)
		}
	}
}

func TestTagsToMap_IAM(t *testing.T) {
	tags := []iamtypes.Tag{
		{Key: aws.String("Application"), Value: aws.String("my-app")},
		{Key: aws.String("Owner"), Value: aws.String("team-a")},
	}

	result := TagsToMap(tags)

	if len(result) != 2 {
		t.Errorf("TagsToMap() returned %d tags, want 2", len(result))
	}

	if result["Application"] != "my-app" {
		t.Errorf("TagsToMap()[Application] = %q, want %q", result["Application"], "my-app")
	}
	if result["Owner"] != "team-a" {
		t.Errorf("TagsToMap()[Owner] = %q, want %q", result["Owner"], "team-a")
	}
}

func TestTagsToMap_S3(t *testing.T) {
	tags := []s3types.Tag{
		{Key: aws.String("BucketType"), Value: aws.String("logs")},
	}

	result := TagsToMap(tags)

	if len(result) != 1 {
		t.Errorf("TagsToMap() returned %d tags, want 1", len(result))
	}

	if result["BucketType"] != "logs" {
		t.Errorf("TagsToMap()[BucketType] = %q, want %q", result["BucketType"], "logs")
	}
}

func TestTagsToMap_Empty(t *testing.T) {
	var tags []ec2types.Tag

	result := TagsToMap(tags)

	if result != nil {
		t.Errorf("TagsToMap(nil) = %v, want nil", result)
	}
}

func TestTagsToMap_NilKey(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: nil, Value: aws.String("value")},
		{Key: aws.String("ValidKey"), Value: aws.String("valid-value")},
	}

	result := TagsToMap(tags)

	// Should only contain the valid tag
	if len(result) != 1 {
		t.Errorf("TagsToMap() returned %d tags, want 1", len(result))
	}

	if result["ValidKey"] != "valid-value" {
		t.Errorf("TagsToMap()[ValidKey] = %q, want %q", result["ValidKey"], "valid-value")
	}
}

func TestTagsToMap_NilValue(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: aws.String("KeyWithNilValue"), Value: nil},
	}

	result := TagsToMap(tags)

	// Key should exist with empty string value
	if len(result) != 1 {
		t.Errorf("TagsToMap() returned %d tags, want 1", len(result))
	}

	if result["KeyWithNilValue"] != "" {
		t.Errorf("TagsToMap()[KeyWithNilValue] = %q, want empty string", result["KeyWithNilValue"])
	}
}
