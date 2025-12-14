package aws

import (
	"testing"
	"time"
)

func TestStr(t *testing.T) {
	tests := []struct {
		name string
		p    *string
		want string
	}{
		{"nil pointer", nil, ""},
		{"empty string", StringPtr(""), ""},
		{"non-empty string", StringPtr("hello"), "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Str(tt.p); got != tt.want {
				t.Errorf("Str() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBool(t *testing.T) {
	tests := []struct {
		name string
		p    *bool
		want bool
	}{
		{"nil pointer", nil, false},
		{"true", BoolPtr(true), true},
		{"false", BoolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Bool(tt.p); got != tt.want {
				t.Errorf("Bool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringPtr(t *testing.T) {
	s := "test"
	p := StringPtr(s)
	if p == nil {
		t.Fatal("StringPtr() returned nil")
	}
	if *p != s {
		t.Errorf("*StringPtr() = %q, want %q", *p, s)
	}
}

func TestBoolPtr(t *testing.T) {
	b := true
	p := BoolPtr(b)
	if p == nil {
		t.Fatal("BoolPtr() returned nil")
	}
	if *p != b {
		t.Errorf("*BoolPtr() = %v, want %v", *p, b)
	}
}

func TestInt32Ptr(t *testing.T) {
	i := int32(42)
	p := Int32Ptr(i)
	if p == nil {
		t.Fatal("Int32Ptr() returned nil")
	}
	if *p != i {
		t.Errorf("*Int32Ptr() = %d, want %d", *p, i)
	}
}

func TestInt32(t *testing.T) {
	tests := []struct {
		name string
		p    *int32
		want int32
	}{
		{"nil pointer", nil, 0},
		{"zero", Int32Ptr(0), 0},
		{"positive", Int32Ptr(42), 42},
		{"negative", Int32Ptr(-1), -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int32(tt.p); got != tt.want {
				t.Errorf("Int32() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestInt64(t *testing.T) {
	int64Ptr := func(i int64) *int64 { return &i }
	tests := []struct {
		name string
		p    *int64
		want int64
	}{
		{"nil pointer", nil, 0},
		{"zero", int64Ptr(0), 0},
		{"positive", int64Ptr(9223372036854775807), 9223372036854775807},
		{"negative", int64Ptr(-1), -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64(tt.p); got != tt.want {
				t.Errorf("Int64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTime(t *testing.T) {
	now := time.Now()
	timePtr := func(t time.Time) *time.Time { return &t }
	tests := []struct {
		name string
		p    *time.Time
		want time.Time
	}{
		{"nil pointer", nil, time.Time{}},
		{"zero time", timePtr(time.Time{}), time.Time{}},
		{"specific time", timePtr(now), now},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Time(tt.p); !got.Equal(tt.want) {
				t.Errorf("Time() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name string
		arn  string
		want string
	}{
		{"empty string", "", ""},
		{"simple role ARN", "arn:aws:iam::123456789012:role/MyRole", "MyRole"},
		{"nested path", "arn:aws:iam::123456789012:role/path/to/MyRole", "MyRole"},
		{"ECS cluster ARN", "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster", "my-cluster"},
		{"S3 bucket ARN", "arn:aws:s3:::my-bucket", "my-bucket"},
		{"Lambda function ARN", "arn:aws:lambda:us-east-1:123456789012:function/my-function", "my-function"},
		{"EC2 instance ARN", "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0", "i-1234567890abcdef0"},
		{"SNS topic ARN", "arn:aws:sns:us-east-1:123456789012:my-topic", "my-topic"},
		{"SQS queue ARN", "arn:aws:sqs:us-east-1:123456789012:my-queue", "my-queue"},
		{"not an ARN", "just-a-name", "just-a-name"},
		{"path with slash", "some/path/resource", "resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractResourceName(tt.arn); got != tt.want {
				t.Errorf("ExtractResourceName(%q) = %q, want %q", tt.arn, got, tt.want)
			}
		})
	}
}
