package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2TagValue(t *testing.T) {
	tags := []types.Tag{
		{Key: aws.String("Name"), Value: aws.String("my-instance")},
		{Key: aws.String("Env"), Value: aws.String("production")},
		{Key: aws.String("Empty"), Value: aws.String("")},
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"existing tag", "Name", "my-instance"},
		{"another tag", "Env", "production"},
		{"empty value", "Empty", ""},
		{"non-existent tag", "NotFound", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EC2TagValue(tags, tt.key); got != tt.want {
				t.Errorf("EC2TagValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEC2TagValue_EmptyTags(t *testing.T) {
	if got := EC2TagValue(nil, "Name"); got != "" {
		t.Errorf("EC2TagValue(nil, Name) = %q, want empty", got)
	}

	if got := EC2TagValue([]types.Tag{}, "Name"); got != "" {
		t.Errorf("EC2TagValue([], Name) = %q, want empty", got)
	}
}

func TestEC2NameTag(t *testing.T) {
	tests := []struct {
		name string
		tags []types.Tag
		want string
	}{
		{
			"has name tag",
			[]types.Tag{{Key: aws.String("Name"), Value: aws.String("my-instance")}},
			"my-instance",
		},
		{
			"no name tag",
			[]types.Tag{{Key: aws.String("Env"), Value: aws.String("prod")}},
			"",
		},
		{
			"empty tags",
			nil,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EC2NameTag(tt.tags); got != tt.want {
				t.Errorf("EC2NameTag() = %q, want %q", got, tt.want)
			}
		})
	}
}
