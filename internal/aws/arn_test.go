package aws

import (
	"testing"
)

func TestParseARN(t *testing.T) {
	tests := []struct {
		name        string
		arn         string
		wantNil     bool
		wantService string
		wantRegion  string
		wantAccount string
		wantResType string
		wantResID   string
	}{
		{
			name:        "EC2 instance",
			arn:         "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			wantService: "ec2",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "instance",
			wantResID:   "i-1234567890abcdef0",
		},
		{
			name:        "EC2 security group",
			arn:         "arn:aws:ec2:us-west-2:123456789012:security-group/sg-12345678",
			wantService: "ec2",
			wantRegion:  "us-west-2",
			wantAccount: "123456789012",
			wantResType: "security-group",
			wantResID:   "sg-12345678",
		},
		{
			name:        "S3 bucket",
			arn:         "arn:aws:s3:::my-bucket",
			wantService: "s3",
			wantRegion:  "",
			wantAccount: "",
			wantResType: "bucket",
			wantResID:   "my-bucket",
		},
		{
			name:        "Lambda function",
			arn:         "arn:aws:lambda:us-east-1:123456789012:function:my-function",
			wantService: "lambda",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "function",
			wantResID:   "my-function",
		},
		{
			name:        "IAM role",
			arn:         "arn:aws:iam::123456789012:role/MyRole",
			wantService: "iam",
			wantRegion:  "",
			wantAccount: "123456789012",
			wantResType: "role",
			wantResID:   "MyRole",
		},
		{
			name:        "IAM policy",
			arn:         "arn:aws:iam::123456789012:policy/MyPolicy",
			wantService: "iam",
			wantRegion:  "",
			wantAccount: "123456789012",
			wantResType: "policy",
			wantResID:   "MyPolicy",
		},
		{
			name:        "ECS cluster",
			arn:         "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster",
			wantService: "ecs",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "cluster",
			wantResID:   "my-cluster",
		},
		{
			name:        "ECS service",
			arn:         "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service",
			wantService: "ecs",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "service",
			wantResID:   "my-cluster/my-service",
		},
		{
			name:        "SNS topic",
			arn:         "arn:aws:sns:us-east-1:123456789012:my-topic",
			wantService: "sns",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "topic",
			wantResID:   "my-topic",
		},
		{
			name:        "SQS queue",
			arn:         "arn:aws:sqs:us-east-1:123456789012:my-queue",
			wantService: "sqs",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "queue",
			wantResID:   "my-queue",
		},
		{
			name:        "DynamoDB table",
			arn:         "arn:aws:dynamodb:us-east-1:123456789012:table/my-table",
			wantService: "dynamodb",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "table",
			wantResID:   "my-table",
		},
		{
			name:        "RDS instance",
			arn:         "arn:aws:rds:us-east-1:123456789012:db:my-database",
			wantService: "rds",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "db",
			wantResID:   "my-database",
		},
		{
			name:        "Step Functions state machine",
			arn:         "arn:aws:states:us-east-1:123456789012:stateMachine:my-state-machine",
			wantService: "states",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "stateMachine",
			wantResID:   "my-state-machine",
		},
		{
			name:        "CloudWatch log group",
			arn:         "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/my-function",
			wantService: "logs",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "log-group",
			wantResID:   "/aws/lambda/my-function",
		},
		{
			name:        "Secrets Manager secret",
			arn:         "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			wantService: "secretsmanager",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "secret",
			wantResID:   "my-secret-AbCdEf",
		},
		{
			name:        "EventBridge event bus",
			arn:         "arn:aws:events:us-east-1:123456789012:event-bus/my-bus",
			wantService: "events",
			wantRegion:  "us-east-1",
			wantAccount: "123456789012",
			wantResType: "event-bus",
			wantResID:   "my-bus",
		},
		{
			name:        "GovCloud partition",
			arn:         "arn:aws-us-gov:ec2:us-gov-west-1:123456789012:instance/i-12345",
			wantService: "ec2",
			wantRegion:  "us-gov-west-1",
			wantAccount: "123456789012",
			wantResType: "instance",
			wantResID:   "i-12345",
		},
		{
			name:    "empty string",
			arn:     "",
			wantNil: true,
		},
		{
			name:    "not an ARN",
			arn:     "not-an-arn",
			wantNil: true,
		},
		{
			name:    "incomplete ARN",
			arn:     "arn:aws:ec2:us-east-1",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseARN(tt.arn)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseARN() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("ParseARN() = nil, want non-nil")
			}

			if got.Service != tt.wantService {
				t.Errorf("Service = %q, want %q", got.Service, tt.wantService)
			}
			if got.Region != tt.wantRegion {
				t.Errorf("Region = %q, want %q", got.Region, tt.wantRegion)
			}
			if got.AccountID != tt.wantAccount {
				t.Errorf("AccountID = %q, want %q", got.AccountID, tt.wantAccount)
			}
			if got.ResourceType != tt.wantResType {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, tt.wantResType)
			}
			if got.ResourceID != tt.wantResID {
				t.Errorf("ResourceID = %q, want %q", got.ResourceID, tt.wantResID)
			}
			if got.Raw != tt.arn {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.arn)
			}
		})
	}
}

func TestARN_ServiceResourceType(t *testing.T) {
	tests := []struct {
		name             string
		arn              string
		wantService      string
		wantResourceType string
	}{
		{
			name:             "EC2 instance",
			arn:              "arn:aws:ec2:us-east-1:123456789012:instance/i-1234",
			wantService:      "ec2",
			wantResourceType: "instances",
		},
		{
			name:             "EC2 security group",
			arn:              "arn:aws:ec2:us-east-1:123456789012:security-group/sg-1234",
			wantService:      "ec2",
			wantResourceType: "security-groups",
		},
		{
			name:             "Lambda function",
			arn:              "arn:aws:lambda:us-east-1:123456789012:function:my-func",
			wantService:      "lambda",
			wantResourceType: "functions",
		},
		{
			name:             "S3 bucket",
			arn:              "arn:aws:s3:::my-bucket",
			wantService:      "s3",
			wantResourceType: "buckets",
		},
		{
			name:             "IAM role",
			arn:              "arn:aws:iam::123456789012:role/MyRole",
			wantService:      "iam",
			wantResourceType: "roles",
		},
		{
			name:             "RDS instance",
			arn:              "arn:aws:rds:us-east-1:123456789012:db:my-db",
			wantService:      "rds",
			wantResourceType: "instances",
		},
		{
			name:             "ECS cluster",
			arn:              "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster",
			wantService:      "ecs",
			wantResourceType: "clusters",
		},
		{
			name:             "Step Functions state machine",
			arn:              "arn:aws:states:us-east-1:123456789012:stateMachine:my-sm",
			wantService:      "stepfunctions",
			wantResourceType: "state-machines",
		},
		{
			name:             "CloudWatch log group",
			arn:              "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/fn",
			wantService:      "cloudwatch",
			wantResourceType: "log-groups",
		},
		{
			name:             "VPC",
			arn:              "arn:aws:ec2:us-east-1:123456789012:vpc/vpc-1234",
			wantService:      "vpc",
			wantResourceType: "vpcs",
		},
		{
			name:             "ELB load balancer",
			arn:              "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/1234",
			wantService:      "elbv2",
			wantResourceType: "load-balancers",
		},
		{
			name:             "EventBridge event bus",
			arn:              "arn:aws:events:us-east-1:123456789012:event-bus/my-bus",
			wantService:      "events",
			wantResourceType: "buses",
		},
		{
			name:             "VPC endpoint",
			arn:              "arn:aws:ec2:us-east-1:123456789012:vpc-endpoint/vpce-1234",
			wantService:      "vpc",
			wantResourceType: "endpoints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseARN(tt.arn)
			if parsed == nil {
				t.Fatalf("ParseARN() returned nil")
			}

			service, resourceType := parsed.ServiceResourceType()
			if service != tt.wantService {
				t.Errorf("service = %q, want %q", service, tt.wantService)
			}
			if resourceType != tt.wantResourceType {
				t.Errorf("resourceType = %q, want %q", resourceType, tt.wantResourceType)
			}
		})
	}
}

func TestARN_ShortID(t *testing.T) {
	tests := []struct {
		name   string
		arn    string
		wantID string
	}{
		{
			name:   "simple ID",
			arn:    "arn:aws:ec2:us-east-1:123456789012:instance/i-1234",
			wantID: "i-1234",
		},
		{
			name:   "nested path",
			arn:    "arn:aws:ecs:us-east-1:123456789012:service/cluster/service-name",
			wantID: "service-name",
		},
		{
			name:   "S3 bucket",
			arn:    "arn:aws:s3:::my-bucket",
			wantID: "my-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseARN(tt.arn)
			if parsed == nil {
				t.Fatalf("ParseARN() returned nil")
			}

			if got := parsed.ShortID(); got != tt.wantID {
				t.Errorf("ShortID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestARN_CanNavigate(t *testing.T) {
	tests := []struct {
		name   string
		arn    string
		canNav bool
	}{
		{
			name:   "EC2 instance - can navigate",
			arn:    "arn:aws:ec2:us-east-1:123456789012:instance/i-1234",
			canNav: true,
		},
		{
			name:   "Lambda function - can navigate",
			arn:    "arn:aws:lambda:us-east-1:123456789012:function:my-func",
			canNav: true,
		},
		{
			name:   "S3 bucket - can navigate",
			arn:    "arn:aws:s3:::my-bucket",
			canNav: true,
		},
		{
			name:   "Unknown service - might navigate (pluralized)",
			arn:    "arn:aws:unknown:us-east-1:123456789012:thing/my-thing",
			canNav: true, // pluralizes to "things"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseARN(tt.arn)
			if parsed == nil {
				t.Fatalf("ParseARN() returned nil")
			}

			if got := parsed.CanNavigate(); got != tt.canNav {
				t.Errorf("CanNavigate() = %v, want %v", got, tt.canNav)
			}
		})
	}
}

func TestParseARN_NilSafe(t *testing.T) {
	var nilARN *ARN

	if nilARN.ShortID() != "" {
		t.Error("ShortID on nil should return empty string")
	}
	if nilARN.String() != "" {
		t.Error("String on nil should return empty string")
	}
	if nilARN.CanNavigate() {
		t.Error("CanNavigate on nil should return false")
	}

	s, r := nilARN.ServiceResourceType()
	if s != "" || r != "" {
		t.Error("ServiceResourceType on nil should return empty strings")
	}

	k, v := nilARN.ExtractParentFilter()
	if k != "" || v != "" {
		t.Error("ExtractParentFilter on nil should return empty strings")
	}
}

func TestARN_ExtractParentFilter(t *testing.T) {
	tests := []struct {
		name      string
		arn       string
		wantKey   string
		wantValue string
	}{
		{
			name:      "GuardDuty finding",
			arn:       "arn:aws:guardduty:us-east-1:123456789012:detector/abc123def456/finding/xyz789",
			wantKey:   "DetectorId",
			wantValue: "abc123def456",
		},
		{
			name:      "Glue table",
			arn:       "arn:aws:glue:us-east-1:123456789012:table/my-database/my-table",
			wantKey:   "DatabaseName",
			wantValue: "my-database",
		},
		{
			name:      "Glue job",
			arn:       "arn:aws:glue:us-east-1:123456789012:job/my-job-name",
			wantKey:   "JobName",
			wantValue: "my-job-name",
		},
		{
			name:      "Transfer user",
			arn:       "arn:aws:transfer:us-east-1:123456789012:user/s-abc123/myuser",
			wantKey:   "ServerId",
			wantValue: "s-abc123",
		},
		{
			name:      "CodeBuild build",
			arn:       "arn:aws:codebuild:us-east-1:123456789012:build/my-project:abc-123-def",
			wantKey:   "ProjectName",
			wantValue: "my-project",
		},
		{
			name:      "EMR cluster",
			arn:       "arn:aws:elasticmapreduce:us-east-1:123456789012:cluster/j-ABC123DEF",
			wantKey:   "ClusterId",
			wantValue: "j-ABC123DEF",
		},
		{
			name:      "ECR repository",
			arn:       "arn:aws:ecr:us-east-1:123456789012:repository/my-repo",
			wantKey:   "RepositoryName",
			wantValue: "my-repo",
		},
		{
			name:      "Cognito user pool",
			arn:       "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_ABC123",
			wantKey:   "UserPoolId",
			wantValue: "us-east-1_ABC123",
		},
		{
			name:      "CodePipeline pipeline",
			arn:       "arn:aws:codepipeline:us-east-1:123456789012:pipeline/my-pipeline",
			wantKey:   "PipelineName",
			wantValue: "my-pipeline",
		},
		{
			name:      "Backup plan",
			arn:       "arn:aws:backup:us-east-1:123456789012:backup-plan/abc-123-def",
			wantKey:   "BackupPlanId",
			wantValue: "abc-123-def",
		},
		{
			name:      "EC2 instance - no parent filter needed",
			arn:       "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			wantKey:   "",
			wantValue: "",
		},
		{
			name:      "Lambda function - no parent filter needed",
			arn:       "arn:aws:lambda:us-east-1:123456789012:function:my-function",
			wantKey:   "",
			wantValue: "",
		},
		{
			name:      "S3 bucket - no parent filter needed",
			arn:       "arn:aws:s3:::my-bucket",
			wantKey:   "",
			wantValue: "",
		},
		{
			name:      "Backup recovery point - cannot extract vault",
			arn:       "arn:aws:backup:us-east-1:123456789012:recovery-point:rp-abc123",
			wantKey:   "",
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseARN(tt.arn)
			if parsed == nil {
				t.Fatalf("ParseARN() returned nil")
			}

			gotKey, gotValue := parsed.ExtractParentFilter()
			if gotKey != tt.wantKey {
				t.Errorf("ExtractParentFilter() key = %q, want %q", gotKey, tt.wantKey)
			}
			if gotValue != tt.wantValue {
				t.Errorf("ExtractParentFilter() value = %q, want %q", gotValue, tt.wantValue)
			}
		})
	}
}
