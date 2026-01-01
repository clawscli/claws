package registry

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ServiceResource uniquely identifies a resource type within an AWS service.
// For example, ServiceResource{Service: "ec2", Resource: "instances"} represents
// the EC2 Instances resource type.
type ServiceResource struct {
	Service  string // AWS service name (e.g., "ec2", "s3", "iam")
	Resource string // Resource type within the service (e.g., "instances", "buckets")
}

func (sr ServiceResource) String() string {
	return fmt.Sprintf("%s/%s", sr.Service, sr.Resource)
}

// Entry holds the factory functions for creating DAO and Renderer instances
// for a specific resource type. Both factories are called lazily when the
// resource is accessed.
type Entry struct {
	DAOFactory      dao.Factory    // Creates a DAO for data access operations
	RendererFactory render.Factory // Creates a Renderer for display formatting
}

// ServiceCategory represents a logical grouping of AWS services for display purposes.
// Categories help organize services in the service browser UI.
type ServiceCategory struct {
	Name     string   // Category display name (e.g., "Compute", "Storage & Database")
	Services []string // Service names in this category, in display order
}

// Registry manages service/resource registrations with priority layers.
// It supports two layers: custom (high priority) and generated (low priority).
// Custom registrations override generated ones for the same service/resource.
//
// The registry also maintains service aliases (e.g., "cfn" -> "cloudformation"),
// display names (e.g., "cloudformation" -> "CloudFormation"), and categories
// for organizing services in the UI.
type Registry struct {
	mu           sync.RWMutex
	custom       map[ServiceResource]Entry // High-priority custom implementations
	generated    map[ServiceResource]Entry // Low-priority generated implementations
	services     map[string][]string       // service -> resource types
	aliases      map[string]string         // alias -> service name or service/resource
	displayNames map[string]string         // service -> display name for UI
	categories   []ServiceCategory         // ordered list of service categories
}

// New creates a new Registry
func New() *Registry {
	return &Registry{
		custom:       make(map[ServiceResource]Entry),
		generated:    make(map[ServiceResource]Entry),
		services:     make(map[string][]string),
		aliases:      defaultAliases(),
		displayNames: defaultDisplayNames(),
		categories:   defaultCategories(),
	}
}

// defaultAliases returns the default service aliases
func defaultAliases() map[string]string {
	return map[string]string{
		"cfn":              "cloudformation",
		"cf":               "cloudformation",
		"sg":               "ec2/security-groups",
		"asg":              "autoscaling",
		"cw":               "cloudwatch",
		"logs":             "cloudwatch/log-groups",
		"r53":              "route53",
		"ssm":              "ssm",
		"sm":               "secretsmanager",
		"ddb":              "dynamodb",
		"sqs":              "sqs",
		"sns":              "sns",
		"agentcore":        "bedrock-agentcore",
		"kb":               "bedrock-agent/knowledge-bases",
		"agent":            "bedrock-agent/agents",
		"models":           "bedrock/foundation-models",
		"guardrail":        "bedrock/guardrails",
		"eb":               "events",
		"eventbridge":      "events",
		"sfn":              "stepfunctions",
		"sq":               "service-quotas",
		"quotas":           "service-quotas",
		"apigw":            "apigateway",
		"api":              "apigateway",
		"elb":              "elbv2",
		"alb":              "elbv2",
		"nlb":              "elbv2",
		"redis":            "elasticache",
		"cache":            "elasticache",
		"es":               "opensearch",
		"elasticsearch":    "opensearch",
		"cdn":              "cloudfront",
		"dist":             "cloudfront",
		"gd":               "guardduty",
		"build":            "codebuild",
		"cb":               "codebuild",
		"pipeline":         "codepipeline",
		"cp":               "codepipeline",
		"waf":              "wafv2",
		"costexplorer":     "ce",
		"cost-explorer":    "ce",
		"ta":               "trustedadvisor",
		"computeoptimizer": "compute-optimizer",
		"co":               "compute-optimizer",
		"sftp":             "transfer",
		"aa":               "accessanalyzer",
		"analyzer":         "accessanalyzer",
		"ri":               "risp/reserved-instances",
		"sp":               "risp/savings-plans",
		"odcr":             "ec2/capacity-reservations",
		"tgw":              "vpc/transit-gateways",
		"cognito":          "cognito-idp",
		"config":           "configservice",
		"macie":            "macie2",
	}
}

// defaultDisplayNames returns the official display names for services
func defaultDisplayNames() map[string]string {
	return map[string]string{
		"accessanalyzer":    "IAM Access Analyzer",
		"acm":               "ACM",
		"apigateway":        "API Gateway",
		"apprunner":         "App Runner",
		"appsync":           "AppSync",
		"athena":            "Athena",
		"autoscaling":       "Auto Scaling",
		"backup":            "AWS Backup",
		"batch":             "Batch",
		"bedrock":           "Bedrock",
		"bedrock-agent":     "Bedrock Agent",
		"bedrock-agentcore": "Bedrock AgentCore",
		"budgets":           "Budgets",
		"cloudformation":    "CloudFormation",
		"cloudfront":        "CloudFront",
		"cloudtrail":        "CloudTrail",
		"cloudwatch":        "CloudWatch",
		"codebuild":         "CodeBuild",
		"codepipeline":      "CodePipeline",
		"cognito-idp":       "Cognito",
		"configservice":     "Config",
		"ce":                "Cost Explorer",
		"datasync":          "DataSync",
		"detective":         "Detective",
		"directconnect":     "Direct Connect",
		"dynamodb":          "DynamoDB",
		"fms":               "Firewall Manager",
		"glue":              "Glue",
		"guardduty":         "GuardDuty",
		"health":            "Health",
		"inspector2":        "Inspector",
		"ec2":               "EC2",
		"ecr":               "ECR",
		"elasticache":       "ElastiCache",
		"ecs":               "ECS",
		"elbv2":             "Elastic Load Balancing",
		"emr":               "EMR",
		"events":            "EventBridge",
		"iam":               "IAM",
		"kinesis":           "Kinesis",
		"kms":               "KMS",
		"lambda":            "Lambda",
		"license-manager":   "License Manager",
		"macie2":            "Macie",
		"network-firewall":  "Network Firewall",
		"opensearch":        "OpenSearch",
		"organizations":     "Organizations",
		"rds":               "RDS",
		"redshift":          "Redshift",
		"risp":              "RI/SP",
		"route53":           "Route 53",
		"s3":                "S3",
		"sagemaker":         "SageMaker",
		"s3vectors":         "S3 Vectors",
		"secretsmanager":    "Secrets Manager",
		"securityhub":       "Security Hub",
		"service-quotas":    "Service Quotas",
		"stepfunctions":     "Step Functions",
		"sns":               "SNS",
		"sqs":               "SQS",
		"ssm":               "Systems Manager",
		"transcribe":        "Transcribe",
		"transfer":          "Transfer Family",
		"vpc":               "VPC",
		"wafv2":             "WAF",
		"xray":              "X-Ray",
		"trustedadvisor":    "Trusted Advisor",
		"compute-optimizer": "Compute Optimizer",
	}
}

// defaultCategories returns the ordered list of service categories
func defaultCategories() []ServiceCategory {
	return []ServiceCategory{
		{
			Name:     "Compute",
			Services: []string{"ec2", "lambda", "ecs", "autoscaling", "apprunner", "batch", "emr"},
		},
		{
			Name:     "Storage & Database",
			Services: []string{"s3", "s3vectors", "dynamodb", "rds", "redshift", "elasticache", "opensearch"},
		},
		{
			Name:     "Containers & ML",
			Services: []string{"ecr", "bedrock", "bedrock-agent", "bedrock-agentcore", "sagemaker", "transcribe"},
		},
		{
			Name:     "Data & Analytics",
			Services: []string{"glue", "athena"},
		},
		{
			Name:     "Networking",
			Services: []string{"vpc", "route53", "apigateway", "appsync", "elbv2", "cloudfront", "directconnect", "network-firewall"},
		},
		{
			Name:     "Security & Identity",
			Services: []string{"iam", "kms", "acm", "secretsmanager", "ssm", "cognito-idp", "guardduty", "wafv2", "inspector2", "securityhub", "fms", "accessanalyzer", "detective", "macie2"},
		},
		{
			Name:     "Integration",
			Services: []string{"sqs", "sns", "events", "stepfunctions", "kinesis", "transfer", "datasync"},
		},
		{
			Name:     "DevOps",
			Services: []string{"codebuild", "codepipeline", "cloudformation"},
		},
		{
			Name:     "Monitoring",
			Services: []string{"cloudwatch", "cloudtrail", "xray", "health"},
		},
		{
			Name:     "Governance",
			Services: []string{"configservice", "organizations", "service-quotas", "license-manager", "backup", "trustedadvisor", "compute-optimizer"},
		},
		{
			Name:     "Cost Management",
			Services: []string{"risp", "ce", "budgets"},
		},
	}
}

// GetDisplayName returns the display name for a service
// Falls back to the service name if no display name is registered
func (r *Registry) GetDisplayName(service string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name, ok := r.displayNames[service]; ok {
		return name
	}
	return service
}

// ResolveAlias resolves an alias to service (and optionally resource)
// Returns (service, resource, found)
func (r *Registry) ResolveAlias(input string) (string, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if resolved, ok := r.aliases[input]; ok {
		// Check if alias includes resource (e.g., "ec2/security-groups")
		for i, c := range resolved {
			if c == '/' {
				return resolved[:i], resolved[i+1:], true
			}
		}
		return resolved, "", true
	}
	return input, "", false
}

// GetAliasesForService returns all aliases for a given service
func (r *Registry) GetAliasesForService(service string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var aliases []string
	for alias, target := range r.aliases {
		// Check if target matches service (handle "service" or "service/resource")
		if target == service || (len(target) > len(service) && target[:len(service)] == service && target[len(service)] == '/') {
			aliases = append(aliases, alias)
		}
	}
	slices.Sort(aliases)
	return aliases
}

// RegisterCustom registers a custom (hand-written) implementation
// Custom implementations take priority over generated ones
func (r *Registry) RegisterCustom(service, resource string, entry Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sr := ServiceResource{Service: service, Resource: resource}
	r.custom[sr] = entry
	r.addService(service, resource)
}

// RegisterGenerated registers a generated implementation
func (r *Registry) RegisterGenerated(service, resource string, entry Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sr := ServiceResource{Service: service, Resource: resource}
	r.generated[sr] = entry
	r.addService(service, resource)
}

func (r *Registry) addService(service, resource string) {
	resources := r.services[service]
	if slices.Contains(resources, resource) {
		return
	}
	r.services[service] = append(resources, resource)
}

// Get retrieves the entry for a service/resource, respecting priority:
// custom > generated
func (r *Registry) Get(service, resource string) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sr := ServiceResource{Service: service, Resource: resource}

	// Priority: custom > generated
	if entry, ok := r.custom[sr]; ok {
		return entry, true
	}
	if entry, ok := r.generated[sr]; ok {
		return entry, true
	}

	return Entry{}, false
}

// HasResource returns true if the service/resource is registered
func (r *Registry) HasResource(service, resource string) bool {
	_, ok := r.Get(service, resource)
	return ok
}

// GetDAO creates a DAO instance for the given service/resource.
// Automatically wraps the DAO for multi-region support if region override is present in context.
func (r *Registry) GetDAO(ctx context.Context, service, resource string) (dao.DAO, error) {
	entry, ok := r.Get(service, resource)
	if !ok {
		return nil, fmt.Errorf("no DAO registered for %s/%s", service, resource)
	}
	if entry.DAOFactory == nil {
		return nil, fmt.Errorf("no DAO factory for %s/%s", service, resource)
	}

	delegate, err := entry.DAOFactory(ctx)
	if err != nil {
		return nil, err
	}

	// Prevent double-wrapping if delegate is already wrapped
	if _, ok := delegate.(*RegionalDAOWrapper); ok {
		return delegate, nil
	}
	if _, ok := delegate.(*PaginatedDAOWrapper); ok {
		return delegate, nil
	}

	// Auto-wrap DAO for multi-region support if region override is present
	if paginated, ok := delegate.(dao.PaginatedDAO); ok {
		return NewPaginatedDAOWrapper(ctx, paginated), nil
	}
	return NewRegionalDAOWrapper(ctx, delegate), nil
}

// GetRenderer creates a Renderer instance for the given service/resource
func (r *Registry) GetRenderer(service, resource string) (render.Renderer, error) {
	entry, ok := r.Get(service, resource)
	if !ok {
		return nil, fmt.Errorf("no renderer registered for %s/%s", service, resource)
	}
	if entry.RendererFactory == nil {
		return nil, fmt.Errorf("no renderer factory for %s/%s", service, resource)
	}
	return entry.RendererFactory(), nil
}

// ListServices returns all registered service names (sorted alphabetically)
func (r *Registry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := slices.Collect(maps.Keys(r.services))
	slices.Sort(services)
	return services
}

// ListServicesByCategory returns services grouped by category
// Only includes services that are actually registered
func (r *Registry) ListServicesByCategory() []ServiceCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ServiceCategory, 0, len(r.categories))
	for _, cat := range r.categories {
		filtered := make([]string, 0, len(cat.Services))
		for _, svc := range cat.Services {
			if _, ok := r.services[svc]; ok {
				filtered = append(filtered, svc)
			}
		}
		if len(filtered) > 0 {
			result = append(result, ServiceCategory{
				Name:     cat.Name,
				Services: filtered,
			})
		}
	}
	return result
}

// ListResources returns all resource types for a service
// Sub-resources (accessible only via navigation) are excluded
func (r *Registry) ListResources(service string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var resources []string
	for _, res := range r.services[service] {
		// Skip sub-resources that are only accessible via navigation
		if isSubResource(service, res) {
			continue
		}
		resources = append(resources, res)
	}

	slices.Sort(resources)
	return resources
}

// subResourceSet contains resources that are only accessible via navigation.
// These resources require a parent context (e.g., stack name, log group name)
// and should only be accessed via navigation from their parent resource.
// Format: "service/resource"
var subResourceSet = map[string]struct{}{
	"cloudformation/events":            {},
	"cloudformation/outputs":           {},
	"cloudformation/resources":         {},
	"cloudwatch/log-streams":           {},
	"service-quotas/quotas":            {},
	"route53/record-sets":              {},
	"apigateway/stages":                {},
	"apigateway/stages-v2":             {},
	"elbv2/targets":                    {},
	"s3vectors/indexes":                {},
	"guardduty/findings":               {},
	"cognito-idp/users":                {},
	"codepipeline/executions":          {},
	"stepfunctions/executions":         {},
	"codebuild/builds":                 {},
	"backup/recovery-points":           {},
	"backup/selections":                {},
	"ecr/images":                       {},
	"autoscaling/activities":           {},
	"bedrock-agent/data-sources":       {},
	"bedrock-agentcore/endpoints":      {},
	"bedrock-agentcore/versions":       {},
	"glue/tables":                      {},
	"glue/job-runs":                    {},
	"athena/query-executions":          {},
	"apprunner/operations":             {},
	"budgets/notifications":            {},
	"vpc/tgw-attachments":              {},
	"directconnect/virtual-interfaces": {},
	"transfer/users":                   {},
	"accessanalyzer/findings":          {},
	"detective/investigations":         {},
	"datasync/task-executions":         {},
	"batch/jobs":                       {},
	"emr/steps":                        {},
	"organizations/ous":                {},
	"license-manager/grants":           {},
	"appsync/data-sources":             {},
	"redshift/snapshots":               {},
}

// isSubResource returns true if the resource is only accessible via navigation
func isSubResource(service, resource string) bool {
	_, ok := subResourceSet[service+"/"+resource]
	return ok
}

// IsSubResource returns true if the resource requires parent context (sub-resource).
// Sub-resources cannot be directly navigated to from tag search results.
func (r *Registry) IsSubResource(service, resource string) bool {
	return isSubResource(service, resource)
}

// Global is the default global registry
var Global = New()
