package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// AWSTag is a constraint for AWS tag types that have Key and Value fields.
type AWSTag interface {
	ec2types.Tag | iamtypes.Tag | s3types.Tag | cfntypes.Tag
}

// tagKeyValue extracts key and value from different AWS tag types.
func tagKeyValue[T AWSTag](tag T) (key, value *string) {
	switch t := any(tag).(type) {
	case ec2types.Tag:
		return t.Key, t.Value
	case iamtypes.Tag:
		return t.Key, t.Value
	case s3types.Tag:
		return t.Key, t.Value
	case cfntypes.Tag:
		return t.Key, t.Value
	}
	return nil, nil
}

// TagsToMap converts any AWS tag slice to a map[string]string.
func TagsToMap[T AWSTag](tags []T) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	m := make(map[string]string, len(tags))
	for _, tag := range tags {
		key, value := tagKeyValue(tag)
		if key != nil {
			m[aws.ToString(key)] = aws.ToString(value)
		}
	}
	return m
}
