package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/clawscli/claws/internal/app"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"

	// Import custom implementations
	_ "github.com/clawscli/claws/custom/ec2/capacityreservations"
	_ "github.com/clawscli/claws/custom/ec2/elasticips"
	_ "github.com/clawscli/claws/custom/ec2/images"
	_ "github.com/clawscli/claws/custom/ec2/instances"
	_ "github.com/clawscli/claws/custom/ec2/keypairs"
	_ "github.com/clawscli/claws/custom/ec2/launchtemplates"
	_ "github.com/clawscli/claws/custom/ec2/securitygroups"
	_ "github.com/clawscli/claws/custom/ec2/snapshots"
	_ "github.com/clawscli/claws/custom/ec2/volumes"
	_ "github.com/clawscli/claws/custom/iam/groups"
	_ "github.com/clawscli/claws/custom/iam/instanceprofiles"
	_ "github.com/clawscli/claws/custom/iam/policies"
	_ "github.com/clawscli/claws/custom/iam/roles"
	_ "github.com/clawscli/claws/custom/iam/users"
	_ "github.com/clawscli/claws/custom/local/profile"
	_ "github.com/clawscli/claws/custom/s3/buckets"
	_ "github.com/clawscli/claws/custom/vpc/internetgateways"
	_ "github.com/clawscli/claws/custom/vpc/natgateways"
	_ "github.com/clawscli/claws/custom/vpc/routetables"
	_ "github.com/clawscli/claws/custom/vpc/subnets"
	_ "github.com/clawscli/claws/custom/vpc/vpcs"

	// RI/SP (Reserved Instances, Savings Plans)
	_ "github.com/clawscli/claws/custom/risp/reservedinstances"
	_ "github.com/clawscli/claws/custom/risp/savingsplans"

	// CloudFormation
	_ "github.com/clawscli/claws/custom/cfn/events"
	_ "github.com/clawscli/claws/custom/cfn/outputs"
	_ "github.com/clawscli/claws/custom/cfn/resources"
	_ "github.com/clawscli/claws/custom/cfn/stacks"

	// DynamoDB
	_ "github.com/clawscli/claws/custom/dynamodb/tables"

	// ECS
	_ "github.com/clawscli/claws/custom/ecs/clusters"
	_ "github.com/clawscli/claws/custom/ecs/services"
	_ "github.com/clawscli/claws/custom/ecs/tasks"

	// Lambda
	_ "github.com/clawscli/claws/custom/lambda/functions"

	// SQS
	_ "github.com/clawscli/claws/custom/sqs/queues"

	// CloudWatch
	_ "github.com/clawscli/claws/custom/cloudwatch/loggroups"
	_ "github.com/clawscli/claws/custom/cloudwatch/logstreams"

	// Secrets Manager
	_ "github.com/clawscli/claws/custom/secretsmanager/secrets"

	// SSM
	_ "github.com/clawscli/claws/custom/ssm/parameters"

	// Bedrock AgentCore
	_ "github.com/clawscli/claws/custom/bedrockagentcore/endpoints"
	_ "github.com/clawscli/claws/custom/bedrockagentcore/runtimes"
	_ "github.com/clawscli/claws/custom/bedrockagentcore/versions"

	// Bedrock Agent
	_ "github.com/clawscli/claws/custom/bedrockagent/agents"
	_ "github.com/clawscli/claws/custom/bedrockagent/datasources"
	_ "github.com/clawscli/claws/custom/bedrockagent/flows"
	_ "github.com/clawscli/claws/custom/bedrockagent/knowledgebases"
	_ "github.com/clawscli/claws/custom/bedrockagent/prompts"

	// Bedrock
	_ "github.com/clawscli/claws/custom/bedrock/foundationmodels"
	_ "github.com/clawscli/claws/custom/bedrock/guardrails"
	_ "github.com/clawscli/claws/custom/bedrock/inferenceprofiles"

	// RDS
	_ "github.com/clawscli/claws/custom/rds/instances"
	_ "github.com/clawscli/claws/custom/rds/snapshots"

	// ECR
	_ "github.com/clawscli/claws/custom/ecr/images"
	_ "github.com/clawscli/claws/custom/ecr/repositories"

	// SNS
	_ "github.com/clawscli/claws/custom/sns/subscriptions"
	_ "github.com/clawscli/claws/custom/sns/topics"

	// EventBridge
	_ "github.com/clawscli/claws/custom/eventbridge/buses"
	_ "github.com/clawscli/claws/custom/eventbridge/rules"

	// Step Functions
	_ "github.com/clawscli/claws/custom/sfn/executions"
	_ "github.com/clawscli/claws/custom/sfn/statemachines"

	// Service Quotas
	_ "github.com/clawscli/claws/custom/servicequotas/quotas"
	_ "github.com/clawscli/claws/custom/servicequotas/services"

	// Route53
	_ "github.com/clawscli/claws/custom/route53/hostedzones"
	_ "github.com/clawscli/claws/custom/route53/recordsets"

	// API Gateway
	_ "github.com/clawscli/claws/custom/apigateway/httpapis"
	_ "github.com/clawscli/claws/custom/apigateway/restapis"
	_ "github.com/clawscli/claws/custom/apigateway/stages"
	_ "github.com/clawscli/claws/custom/apigateway/stagesv2"

	// ELBv2 (ALB/NLB/GLB)
	_ "github.com/clawscli/claws/custom/elbv2/loadbalancers"
	_ "github.com/clawscli/claws/custom/elbv2/targetgroups"
	_ "github.com/clawscli/claws/custom/elbv2/targets"

	// Auto Scaling
	_ "github.com/clawscli/claws/custom/autoscaling/activities"
	_ "github.com/clawscli/claws/custom/autoscaling/groups"

	// KMS
	_ "github.com/clawscli/claws/custom/kms/keys"

	// ACM
	_ "github.com/clawscli/claws/custom/acm/certificates"

	// S3 Vectors
	_ "github.com/clawscli/claws/custom/s3vectors/buckets"
	_ "github.com/clawscli/claws/custom/s3vectors/indexes"

	// ElastiCache
	_ "github.com/clawscli/claws/custom/elasticache/clusters"

	// Kinesis
	_ "github.com/clawscli/claws/custom/kinesis/streams"

	// OpenSearch
	_ "github.com/clawscli/claws/custom/opensearch/domains"

	// CloudFront
	_ "github.com/clawscli/claws/custom/cloudfront/distributions"

	// Cognito
	_ "github.com/clawscli/claws/custom/cognito/userpools"
	_ "github.com/clawscli/claws/custom/cognito/users"

	// GuardDuty
	_ "github.com/clawscli/claws/custom/guardduty/detectors"
	_ "github.com/clawscli/claws/custom/guardduty/findings"

	// CodeBuild
	_ "github.com/clawscli/claws/custom/codebuild/builds"
	_ "github.com/clawscli/claws/custom/codebuild/projects"

	// CodePipeline
	_ "github.com/clawscli/claws/custom/codepipeline/executions"
	_ "github.com/clawscli/claws/custom/codepipeline/pipelines"

	// AWS Backup
	_ "github.com/clawscli/claws/custom/backup/backup-jobs"
	_ "github.com/clawscli/claws/custom/backup/copy-jobs"
	_ "github.com/clawscli/claws/custom/backup/plans"
	_ "github.com/clawscli/claws/custom/backup/protected-resources"
	_ "github.com/clawscli/claws/custom/backup/recovery-points"
	_ "github.com/clawscli/claws/custom/backup/restore-jobs"
	_ "github.com/clawscli/claws/custom/backup/selections"
	_ "github.com/clawscli/claws/custom/backup/vaults"

	// WAF
	_ "github.com/clawscli/claws/custom/wafv2/webacls"

	// Inspector
	_ "github.com/clawscli/claws/custom/inspector2/findings"

	// CloudTrail
	_ "github.com/clawscli/claws/custom/cloudtrail/events"
	_ "github.com/clawscli/claws/custom/cloudtrail/trails"

	// Config
	_ "github.com/clawscli/claws/custom/config/rules"

	// Health
	_ "github.com/clawscli/claws/custom/health/events"

	// X-Ray
	_ "github.com/clawscli/claws/custom/xray/groups"

	// Cost Explorer
	_ "github.com/clawscli/claws/custom/costexplorer/anomalies"
	_ "github.com/clawscli/claws/custom/costexplorer/costs"
	_ "github.com/clawscli/claws/custom/costexplorer/monitors"

	// Trusted Advisor
	_ "github.com/clawscli/claws/custom/trustedadvisor/recommendations"

	// Compute Optimizer
	_ "github.com/clawscli/claws/custom/computeoptimizer/recommendations"
	_ "github.com/clawscli/claws/custom/computeoptimizer/summary"

	// Budgets
	_ "github.com/clawscli/claws/custom/budgets/budgets"
	_ "github.com/clawscli/claws/custom/budgets/notifications"

	// Glue
	_ "github.com/clawscli/claws/custom/glue/crawlers"
	_ "github.com/clawscli/claws/custom/glue/databases"
	_ "github.com/clawscli/claws/custom/glue/jobruns"
	_ "github.com/clawscli/claws/custom/glue/jobs"
	_ "github.com/clawscli/claws/custom/glue/tables"

	// Athena
	_ "github.com/clawscli/claws/custom/athena/queryexecutions"
	_ "github.com/clawscli/claws/custom/athena/workgroups"

	// Security Hub
	_ "github.com/clawscli/claws/custom/securityhub/findings"

	// Firewall Manager
	_ "github.com/clawscli/claws/custom/fms/policies"

	// VPC (Endpoints, Transit Gateways)
	_ "github.com/clawscli/claws/custom/vpc/tgwattachments"
	_ "github.com/clawscli/claws/custom/vpc/transitgateways"
	_ "github.com/clawscli/claws/custom/vpc/vpcendpoints"

	// Network Firewall
	_ "github.com/clawscli/claws/custom/networkfirewall/firewallpolicies"
	_ "github.com/clawscli/claws/custom/networkfirewall/firewalls"
	_ "github.com/clawscli/claws/custom/networkfirewall/rulegroups"

	// Direct Connect
	_ "github.com/clawscli/claws/custom/directconnect/connections"
	_ "github.com/clawscli/claws/custom/directconnect/virtualinterfaces"

	// App Runner
	_ "github.com/clawscli/claws/custom/apprunner/operations"
	_ "github.com/clawscli/claws/custom/apprunner/services"

	// Transcribe
	_ "github.com/clawscli/claws/custom/transcribe/jobs"

	// Transfer Family
	_ "github.com/clawscli/claws/custom/transfer/servers"
	_ "github.com/clawscli/claws/custom/transfer/users"

	// Access Analyzer
	_ "github.com/clawscli/claws/custom/accessanalyzer/analyzers"
	_ "github.com/clawscli/claws/custom/accessanalyzer/findings"

	// Detective
	_ "github.com/clawscli/claws/custom/detective/graphs"
	_ "github.com/clawscli/claws/custom/detective/investigations"

	// DataSync
	_ "github.com/clawscli/claws/custom/datasync/locations"
	_ "github.com/clawscli/claws/custom/datasync/taskexecutions"
	_ "github.com/clawscli/claws/custom/datasync/tasks"

	// Batch
	_ "github.com/clawscli/claws/custom/batch/computeenvironments"
	_ "github.com/clawscli/claws/custom/batch/jobdefinitions"
	_ "github.com/clawscli/claws/custom/batch/jobqueues"
	_ "github.com/clawscli/claws/custom/batch/jobs"

	// EMR
	_ "github.com/clawscli/claws/custom/emr/clusters"
	_ "github.com/clawscli/claws/custom/emr/steps"

	// Organizations
	_ "github.com/clawscli/claws/custom/organizations/accounts"
	_ "github.com/clawscli/claws/custom/organizations/ous"
	_ "github.com/clawscli/claws/custom/organizations/policies"
	_ "github.com/clawscli/claws/custom/organizations/roots"

	// License Manager
	_ "github.com/clawscli/claws/custom/licensemanager/configurations"
	_ "github.com/clawscli/claws/custom/licensemanager/grants"
	_ "github.com/clawscli/claws/custom/licensemanager/licenses"

	// AppSync
	_ "github.com/clawscli/claws/custom/appsync/datasources"
	_ "github.com/clawscli/claws/custom/appsync/graphqlapis"

	// Macie
	_ "github.com/clawscli/claws/custom/macie/buckets"
	_ "github.com/clawscli/claws/custom/macie/classificationjobs"
	_ "github.com/clawscli/claws/custom/macie/findings"

	// Redshift
	_ "github.com/clawscli/claws/custom/redshift/clusters"
	_ "github.com/clawscli/claws/custom/redshift/snapshots"

	// SageMaker
	_ "github.com/clawscli/claws/custom/sagemaker/endpoints"
	_ "github.com/clawscli/claws/custom/sagemaker/models"
	_ "github.com/clawscli/claws/custom/sagemaker/notebooks"
	_ "github.com/clawscli/claws/custom/sagemaker/trainingjobs"
	// Import generated services (uncomment as they are generated)
	// _ "github.com/clawscli/claws/generated/services/ec2"
	// _ "github.com/clawscli/claws/generated/services/s3"
)

// version is set by ldflags during build
var version = "dev"

func main() {
	// Parse command line flags
	opts := parseFlags()

	// Apply CLI options to global config
	cfg := config.Global()

	// Check environment variables (CLI flags take precedence)
	if !opts.readOnly {
		if v := os.Getenv("CLAWS_READ_ONLY"); v == "1" || v == "true" {
			opts.readOnly = true
		}
	}
	cfg.SetReadOnly(opts.readOnly)
	if opts.envCreds {
		// Use environment credentials, ignore ~/.aws config
		cfg.UseEnvOnly()
	} else if opts.profile != "" {
		cfg.UseProfile(opts.profile)
		os.Setenv("AWS_PROFILE", opts.profile)
	}
	// else: SDKDefault is the zero value, no action needed
	if opts.region != "" {
		cfg.SetRegion(opts.region)
		os.Setenv("AWS_REGION", opts.region)
	}

	// Enable logging if log file specified
	if opts.logFile != "" {
		if err := log.EnableFile(opts.logFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file %s: %v\n", opts.logFile, err)
		} else {
			log.Info("claws started", "profile", opts.profile, "region", opts.region, "readOnly", opts.readOnly)
		}
	}

	ctx := context.Background()

	// Create the application
	application := app.New(ctx, registry.Global)

	// Run the TUI
	// Note: In v2, AltScreen and MouseMode are set via the View struct
	// v2 has better ESC key handling via x/input package
	p := tea.NewProgram(application)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cliOptions holds command line options
type cliOptions struct {
	profile  string
	region   string
	readOnly bool
	envCreds bool
	logFile  string
}

// parseFlags parses command line flags and returns options
func parseFlags() cliOptions {
	opts := cliOptions{}
	showHelp := false
	showVersion := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-p" || arg == "--profile":
			if i+1 < len(args) {
				i++
				opts.profile = args[i]
			}
		case arg == "-r" || arg == "--region":
			if i+1 < len(args) {
				i++
				opts.region = args[i]
			}
		case arg == "-ro" || arg == "--read-only":
			opts.readOnly = true
		case arg == "-e" || arg == "--env":
			opts.envCreds = true
		case arg == "-l" || arg == "--log-file":
			if i+1 < len(args) {
				i++
				opts.logFile = args[i]
			}
		case arg == "-h" || arg == "--help":
			showHelp = true
		case arg == "-v" || arg == "--version":
			showVersion = true
		}
	}

	if showVersion {
		fmt.Printf("claws %s\n", version)
		os.Exit(0)
	}

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	return opts
}

func printUsage() {
	fmt.Println("claws - A terminal UI for AWS resource management")
	fmt.Println()
	fmt.Println("Usage: claws [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -p, --profile <name>")
	fmt.Println("        AWS profile to use")
	fmt.Println("  -r, --region <region>")
	fmt.Println("        AWS region to use")
	fmt.Println("  -e, --env")
	fmt.Println("        Use environment credentials (ignore ~/.aws config)")
	fmt.Println("        Useful for instance profiles, ECS task roles, Lambda, etc.")
	fmt.Println("  -ro, --read-only")
	fmt.Println("        Run in read-only mode (disable dangerous actions)")
	fmt.Println("  -l, --log-file <path>")
	fmt.Println("        Enable debug logging to specified file")
	fmt.Println("  -v, --version")
	fmt.Println("        Show version")
	fmt.Println("  -h, --help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CLAWS_READ_ONLY=1|true   Enable read-only mode")
}
