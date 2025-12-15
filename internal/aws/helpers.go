package aws

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Str safely dereferences a string pointer, returning empty string if nil.
// This is an alias for aws.ToString for convenience.
func Str(p *string) string {
	return aws.ToString(p)
}

// Bool safely dereferences a bool pointer, returning false if nil.
func Bool(p *bool) bool {
	return aws.ToBool(p)
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return aws.String(s)
}

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool {
	return aws.Bool(b)
}

// Int32 safely dereferences an int32 pointer, returning 0 if nil.
func Int32(p *int32) int32 {
	return aws.ToInt32(p)
}

// Int64 safely dereferences an int64 pointer, returning 0 if nil.
func Int64(p *int64) int64 {
	return aws.ToInt64(p)
}

// Float64 safely dereferences a float64 pointer, returning 0 if nil.
func Float64(p *float64) float64 {
	return aws.ToFloat64(p)
}

// Time safely dereferences a time.Time pointer, returning zero time if nil.
func Time(p *time.Time) time.Time {
	return aws.ToTime(p)
}

// Int32Ptr returns a pointer to the given int32.
func Int32Ptr(i int32) *int32 {
	return aws.Int32(i)
}

// ExtractResourceName extracts the resource name from an AWS ARN.
// e.g., "arn:aws:iam::123456789012:role/MyRole" -> "MyRole"
// e.g., "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster" -> "my-cluster"
// e.g., "arn:aws:s3:::my-bucket" -> "my-bucket"
func ExtractResourceName(arn string) string {
	if arn == "" {
		return ""
	}

	// Handle simple ARN format with "/" separator
	if idx := strings.LastIndex(arn, "/"); idx != -1 {
		return arn[idx+1:]
	}

	// Handle ARN format with ":" separator (e.g., S3 buckets)
	if strings.HasPrefix(arn, "arn:aws:") {
		parts := strings.Split(arn, ":")
		if len(parts) >= 6 {
			return parts[len(parts)-1]
		}
	}

	return arn
}
