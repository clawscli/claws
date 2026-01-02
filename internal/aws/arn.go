package aws

import (
	"strings"

	sdkarn "github.com/aws/aws-sdk-go-v2/aws/arn"
)

// ARN represents a parsed Amazon Resource Name with additional resource type/ID extraction.
// Wraps the AWS SDK's arn.ARN and adds ResourceType/ResourceID parsing for claws navigation.
type ARN struct {
	Partition    string // aws, aws-cn, aws-us-gov
	Service      string // ec2, s3, lambda, etc.
	Region       string // us-east-1, etc. (empty for global resources like IAM, S3)
	AccountID    string // 123456789012 (empty for some resources like S3 buckets)
	ResourceType string // instance, bucket, function, etc.
	ResourceID   string // i-1234567890abcdef0, my-bucket, etc.
	Raw          string // original ARN string
}

// ParseARN parses an ARN string into its components using the AWS SDK.
// Returns nil if the string is not a valid ARN.
func ParseARN(arn string) *ARN {
	if !sdkarn.IsARN(arn) {
		return nil
	}

	parsed, err := sdkarn.Parse(arn)
	if err != nil {
		return nil
	}

	a := &ARN{
		Partition: parsed.Partition,
		Service:   parsed.Service,
		Region:    parsed.Region,
		AccountID: parsed.AccountID,
		Raw:       arn,
	}

	// Parse the resource part into ResourceType and ResourceID
	a.parseResource(parsed.Resource)

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
// Maps ARN service and resource types to claws registry naming conventions.
func (a *ARN) ServiceResourceType() (service, resourceType string) {
	if a == nil {
		return "", ""
	}

	resourceType = normalizeResourceType(a.Service, a.ResourceType)
	service = normalizeService(a.Service, a.ResourceType)

	return service, resourceType
}

func normalizeService(arnService, resourceType string) string {
	if arnService == "ec2" && isVPCResource(resourceType) {
		return "vpc"
	}
	if mapped, ok := arnToRegistryService[arnService]; ok {
		return mapped
	}
	return arnService
}

func isVPCResource(resourceType string) bool {
	switch resourceType {
	case "vpc", "subnet", "route-table", "internet-gateway", "nat-gateway", "vpc-endpoint", "transit-gateway":
		return true
	}
	return false
}

var arnToRegistryService = map[string]string{
	"logs":                 "cloudwatch",
	"states":               "stepfunctions",
	"elasticloadbalancing": "elbv2",
	"execute-api":          "apigateway",
	"config":               "configservice",
	"access-analyzer":      "accessanalyzer",
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

var arnToRegistryType = map[string]string{
	"ec2/instance":                      "instances",
	"ec2/volume":                        "volumes",
	"ec2/security-group":                "security-groups",
	"ec2/elastic-ip":                    "elastic-ips",
	"ec2/key-pair":                      "key-pairs",
	"ec2/image":                         "images",
	"ec2/snapshot":                      "snapshots",
	"ec2/launch-template":               "launch-templates",
	"ec2/capacity-reservation":          "capacity-reservations",
	"ec2/vpc":                           "vpcs",
	"ec2/subnet":                        "subnets",
	"ec2/route-table":                   "route-tables",
	"ec2/internet-gateway":              "internet-gateways",
	"ec2/nat-gateway":                   "nat-gateways",
	"ec2/vpc-endpoint":                  "endpoints",
	"ec2/transit-gateway":               "transit-gateways",
	"lambda/function":                   "functions",
	"ecs/cluster":                       "clusters",
	"ecs/service":                       "services",
	"ecs/task":                          "tasks",
	"ecs/task-definition":               "task-definitions",
	"ecs/container-instance":            "container-instances",
	"s3/bucket":                         "buckets",
	"rds/db":                            "instances",
	"rds/cluster":                       "clusters",
	"rds/snapshot":                      "snapshots",
	"iam/user":                          "users",
	"iam/role":                          "roles",
	"iam/policy":                        "policies",
	"iam/group":                         "groups",
	"iam/instance-profile":              "instance-profiles",
	"dynamodb/table":                    "tables",
	"sns/topic":                         "topics",
	"sqs/queue":                         "queues",
	"logs/log-group":                    "log-groups",
	"states/stateMachine":               "state-machines",
	"states/execution":                  "executions",
	"secretsmanager/secret":             "secrets",
	"kms/key":                           "keys",
	"events/event-bus":                  "buses",
	"events/rule":                       "rules",
	"apigateway/restapis":               "rest-apis",
	"cloudformation/stack":              "stacks",
	"autoscaling/autoScalingGroup":      "groups",
	"elasticloadbalancing/loadbalancer": "load-balancers",
	"elasticloadbalancing/targetgroup":  "target-groups",
	"elasticloadbalancing/app":          "load-balancers",
	"elasticloadbalancing/net":          "load-balancers",
	"ecr/repository":                    "repositories",
	"kinesis/stream":                    "streams",
	"glue/database":                     "databases",
	"glue/table":                        "tables",
	"glue/crawler":                      "crawlers",
	"glue/job":                          "jobs",
	"bedrock/foundation-model":          "foundation-models",
	"bedrock/inference-profile":         "inference-profiles",
	"bedrock/guardrail":                 "guardrails",
	"bedrock-agent/agent":               "agents",
	"bedrock-agent/knowledge-base":      "knowledge-bases",
	"bedrock-agent/flow":                "flows",
	"bedrock-agentcore/runtime":         "runtimes",
	"route53/hostedzone":                "hosted-zones",
	"cloudfront/distribution":           "distributions",
	"acm/certificate":                   "certificates",
	"ssm/parameter":                     "parameters",
	"cognito-idp/userpool":              "user-pools",
	"guardduty/detector":                "detectors",
	"config/config-rule":                "rules",
	"backup/backup-vault":               "vaults",
	"backup/backup-plan":                "plans",
	"organizations/account":             "accounts",
	"organizations/ou":                  "ous",
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

// ExtractParentFilter returns the filter key and value needed for sub-resources
// that require parent context when calling Get(). Returns empty strings if no
// parent filter is needed or cannot be extracted from the ARN.
//
// Example: For guardduty finding ARN "arn:aws:guardduty:us-east-1:123:detector/abc/finding/xyz"
// returns ("DetectorId", "abc")
func (a *ARN) ExtractParentFilter() (key, value string) {
	if a == nil || a.ResourceID == "" {
		return "", ""
	}

	// Build lookup key from ARN service and resource type
	lookupKey := a.Service + "/" + a.ResourceType

	switch lookupKey {
	// GuardDuty findings: detector/detector-id/finding/finding-id
	case "guardduty/detector":
		// ResourceID = "detector-id/finding/finding-id"
		if idx := strings.Index(a.ResourceID, "/finding/"); idx != -1 {
			return "DetectorId", a.ResourceID[:idx]
		}

	// Glue tables: table/database-name/table-name
	case "glue/table":
		// ResourceID = "database-name/table-name"
		if idx := strings.Index(a.ResourceID, "/"); idx != -1 {
			return "DatabaseName", a.ResourceID[:idx]
		}

	// Glue job runs: job/job-name (run ID not in ARN)
	case "glue/job":
		// ResourceID = "job-name" or "job-name/run/run-id"
		id := a.ResourceID
		if idx := strings.Index(id, "/"); idx != -1 {
			return "JobName", id[:idx]
		}
		return "JobName", id

	// Transfer users: user/server-id/user-name
	case "transfer/user":
		// ResourceID = "server-id/user-name"
		if idx := strings.Index(a.ResourceID, "/"); idx != -1 {
			return "ServerId", a.ResourceID[:idx]
		}

	// CodeBuild builds: build/project-name:build-id
	case "codebuild/build":
		// ResourceID = "project-name:build-id"
		if idx := strings.Index(a.ResourceID, ":"); idx != -1 {
			return "ProjectName", a.ResourceID[:idx]
		}

	// CodePipeline executions: pipeline-name (execution ID separate)
	case "codepipeline/pipeline":
		// ResourceID could be "pipeline-name" or include execution info
		id := a.ResourceID
		if idx := strings.Index(id, "/"); idx != -1 {
			return "PipelineName", id[:idx]
		}
		return "PipelineName", id

	// EMR steps: cluster/cluster-id (step ID not in standard ARN)
	case "elasticmapreduce/cluster":
		// ResourceID = "cluster-id" or "cluster-id/step/step-id"
		id := a.ResourceID
		if idx := strings.Index(id, "/"); idx != -1 {
			return "ClusterId", id[:idx]
		}
		return "ClusterId", id

	// ECR images: repository/repo-name
	case "ecr/repository":
		// ResourceID = "repo-name" or "repo-name/image/sha256:..."
		id := a.ResourceID
		if idx := strings.Index(id, "/"); idx != -1 {
			return "RepositoryName", id[:idx]
		}
		return "RepositoryName", id

	// Cognito users: userpool/pool-id
	case "cognito-idp/userpool":
		// ResourceID = "pool-id" or "pool-id/user/username"
		id := a.ResourceID
		if idx := strings.Index(id, "/"); idx != -1 {
			return "UserPoolId", id[:idx]
		}
		return "UserPoolId", id

	// Access Analyzer findings - need analyzer ARN
	case "access-analyzer/analyzer":
		// ResourceID = "analyzer-name/finding/finding-id"
		if idx := strings.Index(a.ResourceID, "/finding/"); idx != -1 {
			// Return the full analyzer ARN
			return "AnalyzerArn", "arn:" + a.Partition + ":access-analyzer:" + a.Region + ":" + a.AccountID + ":analyzer/" + a.ResourceID[:idx]
		}

	// Detective investigations - need graph ARN
	case "detective/graph":
		// ResourceID = "graph-id/investigation/inv-id"
		if idx := strings.Index(a.ResourceID, "/investigation/"); idx != -1 {
			return "GraphArn", "arn:" + a.Partition + ":detective:" + a.Region + ":" + a.AccountID + ":graph:" + a.ResourceID[:idx]
		}

	// Backup recovery points - vault name in ARN path
	case "backup/recovery-point":
		// ARN format varies: arn:aws:backup:region:account:recovery-point:rp-id
		// The vault name is NOT in the ARN for recovery points
		// Cannot extract parent filter
		return "", ""

	// Backup selections - plan ID not in ARN
	case "backup/backup-plan":
		// ResourceID = "plan-id" or "plan-id/selection/selection-id"
		id := a.ResourceID
		if idx := strings.Index(id, "/"); idx != -1 {
			return "BackupPlanId", id[:idx]
		}
		return "BackupPlanId", id
	}

	return "", ""
}
