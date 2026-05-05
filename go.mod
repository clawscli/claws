module github.com/clawscli/claws

go 1.25.4

require (
	charm.land/bubbles/v2 v2.1.0
	charm.land/bubbletea/v2 v2.0.6
	charm.land/lipgloss/v2 v2.0.3
	github.com/atotto/clipboard v0.1.4
	github.com/aws/aws-sdk-go-v2 v1.41.7
	github.com/aws/aws-sdk-go-v2/config v1.32.17
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.47.2
	github.com/aws/aws-sdk-go-v2/service/acm v1.38.3
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.39.3
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.34.3
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.39.16
	github.com/aws/aws-sdk-go-v2/service/appsync v1.53.7
	github.com/aws/aws-sdk-go-v2/service/athena v1.57.6
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.66.2
	github.com/aws/aws-sdk-go-v2/service/backup v1.55.2
	github.com/aws/aws-sdk-go-v2/service/batch v1.64.1
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.59.2
	github.com/aws/aws-sdk-go-v2/service/bedrockagent v1.53.2
	github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol v1.35.0
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.50.6
	github.com/aws/aws-sdk-go-v2/service/budgets v1.43.6
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.11
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.62.0
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.55.11
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.57.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.73.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.68.15
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.46.23
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.60.2
	github.com/aws/aws-sdk-go-v2/service/computeoptimizer v1.50.1
	github.com/aws/aws-sdk-go-v2/service/configservice v1.62.3
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.63.8
	github.com/aws/aws-sdk-go-v2/service/datasync v1.58.4
	github.com/aws/aws-sdk-go-v2/service/detective v1.38.15
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.38.17
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.3
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.300.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.57.2
	github.com/aws/aws-sdk-go-v2/service/ecs v1.79.1
	github.com/aws/aws-sdk-go-v2/service/eks v1.83.0
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.52.2
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.12
	github.com/aws/aws-sdk-go-v2/service/emr v1.59.2
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.45.25
	github.com/aws/aws-sdk-go-v2/service/fms v1.44.24
	github.com/aws/aws-sdk-go-v2/service/gamelift v1.54.0
	github.com/aws/aws-sdk-go-v2/service/glue v1.140.1
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.75.3
	github.com/aws/aws-sdk-go-v2/service/health v1.37.6
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.10
	github.com/aws/aws-sdk-go-v2/service/inspector2 v1.47.6
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.43.7
	github.com/aws/aws-sdk-go-v2/service/kms v1.51.1
	github.com/aws/aws-sdk-go-v2/service/lambda v1.90.1
	github.com/aws/aws-sdk-go-v2/service/licensemanager v1.37.12
	github.com/aws/aws-sdk-go-v2/service/macie2 v1.51.2
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.60.1
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.67.1
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.3
	github.com/aws/aws-sdk-go-v2/service/rds v1.118.2
	github.com/aws/aws-sdk-go-v2/service/redshift v1.62.7
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.31.12
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.100.1
	github.com/aws/aws-sdk-go-v2/service/s3vectors v1.6.8
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.244.0
	github.com/aws/aws-sdk-go-v2/service/savingsplans v1.32.4
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.7
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.69.2
	github.com/aws/aws-sdk-go-v2/service/servicequotas v1.34.7
	github.com/aws/aws-sdk-go-v2/service/sfn v1.40.12
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.17
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.27
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.6
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1
	github.com/aws/aws-sdk-go-v2/service/transcribe v1.54.6
	github.com/aws/aws-sdk-go-v2/service/transfer v1.72.0
	github.com/aws/aws-sdk-go-v2/service/trustedadvisor v1.14.6
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.71.5
	github.com/aws/aws-sdk-go-v2/service/xray v1.36.23
	github.com/aws/smithy-go v1.25.1
	github.com/charmbracelet/x/ansi v0.11.7
	github.com/creack/pty v1.1.24
	github.com/google/uuid v1.6.0
	github.com/mattn/go-runewidth v0.0.23
	golang.org/x/sync v0.20.0
	golang.org/x/term v0.42.0
	gopkg.in/ini.v1 v1.67.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260428153724-66037269d7be // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/sys v0.43.0 // indirect
)
