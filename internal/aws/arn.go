package aws

import (
	"strings"
)

// ARN represents a parsed Amazon Resource Name.
// Format: arn:partition:service:region:account-id:resource
// Resource can be: resource-type/resource-id, resource-type:resource-id, or just resource
type ARN struct {
	Partition    string // aws, aws-cn, aws-us-gov
	Service      string // ec2, s3, lambda, etc.
	Region       string // us-east-1, etc. (empty for global resources like IAM, S3)
	AccountID    string // 123456789012 (empty for some resources like S3 buckets)
	ResourceType string // instance, bucket, function, etc.
	ResourceID   string // i-1234567890abcdef0, my-bucket, etc.
	Raw          string // original ARN string
}

// ParseARN parses an ARN string into its components.
// Returns nil if the string is not a valid ARN.
func ParseARN(arn string) *ARN {
	if arn == "" || !strings.HasPrefix(arn, "arn:") {
		return nil
	}

	// Split by colon: arn:partition:service:region:account:resource
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return nil
	}

	a := &ARN{
		Partition: parts[1],
		Service:   parts[2],
		Region:    parts[3],
		AccountID: parts[4],
		Raw:       arn,
	}

	// Parse the resource part (parts[5])
	resource := parts[5]
	a.parseResource(resource)

	return a
}

// parseResource extracts resourceType and resourceID from the resource portion.
// Handles various formats:
// - resource-type/resource-id (most common: ec2, ecs, lambda)
// - resource-type:resource-id (some services like sns, sqs, logs)
// - just resource-id (s3 buckets, simple resources)
// - resource-type/sub-type/resource-id (nested paths)
func (a *ARN) parseResource(resource string) {
	if resource == "" {
		return
	}

	// Find both separators
	slashIdx := strings.Index(resource, "/")
	colonIdx := strings.Index(resource, ":")

	// Use the first separator found
	// Special case: if colon comes before slash (like log-group:/aws/...),
	// the colon is the separator
	switch {
	case colonIdx != -1 && (slashIdx == -1 || colonIdx < slashIdx):
		// Colon separator (lambda:function:name, log-group:/path, etc.)
		a.ResourceType = resource[:colonIdx]
		a.ResourceID = resource[colonIdx+1:]
	case slashIdx != -1:
		// Slash separator (instance/i-1234, cluster/name, etc.)
		a.ResourceType = resource[:slashIdx]
		a.ResourceID = resource[slashIdx+1:]
	default:
		// No separator - the whole thing is the resource ID
		// This happens with S3 buckets, simple resources
		a.ResourceID = resource
		// Infer resource type from service for common cases
		a.ResourceType = inferResourceType(a.Service, resource)
	}
}

// inferResourceType attempts to determine the resource type when not explicit in ARN.
func inferResourceType(service, resource string) string {
	switch service {
	case "s3":
		return "bucket"
	case "sns":
		return "topic"
	case "sqs":
		return "queue"
	case "dynamodb":
		return "table"
	case "events":
		return "event-bus"
	default:
		return ""
	}
}

// ShortID returns a shortened version of ResourceID for display.
// Strips common prefixes and truncates long IDs.
func (a *ARN) ShortID() string {
	if a == nil || a.ResourceID == "" {
		return ""
	}

	id := a.ResourceID

	// For nested paths, use the last segment
	if idx := strings.LastIndex(id, "/"); idx != -1 {
		id = id[idx+1:]
	}

	return id
}

// String returns the original ARN string.
func (a *ARN) String() string {
	if a == nil {
		return ""
	}
	return a.Raw
}

// ServiceResourceType returns a normalized service/resource-type pair for registry lookup.
// Maps ARN resource types to claws registry resource types.
func (a *ARN) ServiceResourceType() (service, resourceType string) {
	if a == nil {
		return "", ""
	}

	service = a.Service
	resourceType = normalizeResourceType(a.Service, a.ResourceType)

	return service, resourceType
}

// normalizeResourceType maps ARN resource types to claws registry resource types.
// ARN types are often singular, registry uses plural.
func normalizeResourceType(service, arnType string) string {
	// Service-specific mappings
	key := service + "/" + arnType
	if mapped, ok := arnToRegistryType[key]; ok {
		return mapped
	}

	// Default: just pluralize common patterns
	if arnType == "" {
		return ""
	}

	// Simple pluralization for common cases
	switch {
	case strings.HasSuffix(arnType, "s"):
		return arnType // already plural
	case strings.HasSuffix(arnType, "y"):
		return arnType[:len(arnType)-1] + "ies"
	default:
		return arnType + "s"
	}
}

// arnToRegistryType maps "service/arn-type" to claws registry resourceType.
// Only include mappings that differ from simple pluralization.
var arnToRegistryType = map[string]string{
	// EC2
	"ec2/instance":             "instances",
	"ec2/volume":               "volumes",
	"ec2/security-group":       "securitygroups",
	"ec2/elastic-ip":           "elasticips",
	"ec2/key-pair":             "keypairs",
	"ec2/image":                "images",
	"ec2/snapshot":             "snapshots",
	"ec2/launch-template":      "launchtemplates",
	"ec2/capacity-reservation": "capacityreservations",
	"ec2/vpc":                  "vpcs",
	"ec2/subnet":               "subnets",
	"ec2/route-table":          "routetables",
	"ec2/internet-gateway":     "internetgateways",
	"ec2/nat-gateway":          "natgateways",
	"ec2/vpc-endpoint":         "vpcendpoints",
	"ec2/transit-gateway":      "transitgateways",

	// Lambda
	"lambda/function": "functions",

	// ECS
	"ecs/cluster":                   "clusters",
	"ecs/service":                   "services",
	"ecs/task":                      "tasks",
	"ecs/task-definition":           "taskdefinitions",
	"ecs/container-instance":        "containerinstances",
	"ecs/capacity-provider":         "capacityproviders",
	"ecs/task-set":                  "tasksets",
	"ecs/cluster-capacity-provider": "clustercapacityproviders",

	// S3
	"s3/bucket": "buckets",

	// RDS
	"rds/db":       "instances",
	"rds/cluster":  "clusters",
	"rds/snapshot": "snapshots",

	// IAM
	"iam/user":             "users",
	"iam/role":             "roles",
	"iam/policy":           "policies",
	"iam/group":            "groups",
	"iam/instance-profile": "instanceprofiles",

	// DynamoDB
	"dynamodb/table": "tables",

	// SNS
	"sns/topic": "topics",

	// SQS
	"sqs/queue": "queues",

	// CloudWatch
	"logs/log-group": "loggroups",

	// Step Functions
	"states/stateMachine": "statemachines",
	"states/execution":    "executions",

	// Secrets Manager
	"secretsmanager/secret": "secrets",

	// KMS
	"kms/key": "keys",

	// EventBridge
	"events/event-bus": "buses",
	"events/rule":      "rules",

	// API Gateway
	"apigateway/restapis": "restapis",
	"execute-api/":        "restapis",

	// CloudFormation
	"cloudformation/stack": "stacks",

	// Auto Scaling
	"autoscaling/autoScalingGroup": "groups",

	// ELB
	"elasticloadbalancing/loadbalancer":  "loadbalancers",
	"elasticloadbalancing/targetgroup":   "targetgroups",
	"elasticloadbalancing/app":           "loadbalancers",
	"elasticloadbalancing/net":           "loadbalancers",
	"elasticloadbalancing/listener":      "listeners",
	"elasticloadbalancing/listener-rule": "listenerrules",

	// ECR
	"ecr/repository": "repositories",

	// Kinesis
	"kinesis/stream": "streams",

	// Glue
	"glue/database": "databases",
	"glue/table":    "tables",
	"glue/crawler":  "crawlers",
	"glue/job":      "jobs",

	// Bedrock
	"bedrock/foundation-model":     "foundationmodels",
	"bedrock/inference-profile":    "inferenceprofiles",
	"bedrock/guardrail":            "guardrails",
	"bedrock-agent/agent":          "agents",
	"bedrock-agent/knowledge-base": "knowledgebases",
	"bedrock-agent/flow":           "flows",

	// Route 53
	"route53/hostedzone": "hostedzones",

	// CloudFront
	"cloudfront/distribution": "distributions",

	// ACM
	"acm/certificate": "certificates",

	// SSM
	"ssm/parameter": "parameters",

	// Cognito
	"cognito-idp/userpool": "userpools",

	// GuardDuty
	"guardduty/detector": "detectors",

	// Security Hub
	"securityhub/hub": "hubs",

	// Config
	"config/config-rule": "rules",

	// Backup
	"backup/backup-vault": "vaults",
	"backup/backup-plan":  "plans",

	// Organizations
	"organizations/account": "accounts",
	"organizations/ou":      "ous",
}

// CanNavigate returns true if this ARN can be navigated to in claws.
func (a *ARN) CanNavigate() bool {
	if a == nil {
		return false
	}
	service, resType := a.ServiceResourceType()
	if service == "" || resType == "" {
		return false
	}
	// Check if we have a mapping (explicit or derived)
	key := service + "/" + a.ResourceType
	if _, ok := arnToRegistryType[key]; ok {
		return true
	}
	// Allow navigation for common patterns even without explicit mapping
	return resType != ""
}
