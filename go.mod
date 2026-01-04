module github.com/clawscli/claws

go 1.25.4

require (
	charm.land/bubbles/v2 v2.0.0-rc.1
	charm.land/bubbletea/v2 v2.0.0-rc.2
	charm.land/lipgloss/v2 v2.0.0-beta.3.0.20251106192539-4b304240aab7
	github.com/atotto/clipboard v0.1.4
	github.com/aws/aws-sdk-go-v2 v1.41.0
	github.com/aws/aws-sdk-go-v2/config v1.32.5
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.45.7
	github.com/aws/aws-sdk-go-v2/service/acm v1.37.18
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.38.3
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.33.4
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.39.9
	github.com/aws/aws-sdk-go-v2/service/appsync v1.53.0
	github.com/aws/aws-sdk-go-v2/service/athena v1.56.4
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.62.4
	github.com/aws/aws-sdk-go-v2/service/backup v1.54.5
	github.com/aws/aws-sdk-go-v2/service/batch v1.58.11
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.53.0
	github.com/aws/aws-sdk-go-v2/service/bedrockagent v1.52.2
	github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol v1.15.1
	github.com/aws/aws-sdk-go-v2/service/budgets v1.42.3
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.4
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.58.3
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.55.4
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.53.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.62.2
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.68.8
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.46.16
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.57.17
	github.com/aws/aws-sdk-go-v2/service/computeoptimizer v1.49.3
	github.com/aws/aws-sdk-go-v2/service/configservice v1.59.9
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.62.0
	github.com/aws/aws-sdk-go-v2/service/datasync v1.57.0
	github.com/aws/aws-sdk-go-v2/service/detective v1.38.8
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.38.10
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.53.5
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.276.1
	github.com/aws/aws-sdk-go-v2/service/ecr v1.54.4
	github.com/aws/aws-sdk-go-v2/service/ecs v1.69.5
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.51.8
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.5
	github.com/aws/aws-sdk-go-v2/service/emr v1.57.4
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.45.17
	github.com/aws/aws-sdk-go-v2/service/fms v1.44.16
	github.com/aws/aws-sdk-go-v2/service/glue v1.135.3
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.70.1
	github.com/aws/aws-sdk-go-v2/service/health v1.35.5
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.1
	github.com/aws/aws-sdk-go-v2/service/inspector2 v1.46.1
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.42.9
	github.com/aws/aws-sdk-go-v2/service/kms v1.49.4
	github.com/aws/aws-sdk-go-v2/service/lambda v1.87.0
	github.com/aws/aws-sdk-go-v2/service/licensemanager v1.37.4
	github.com/aws/aws-sdk-go-v2/service/macie2 v1.50.8
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.59.2
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.56.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.50.0
	github.com/aws/aws-sdk-go-v2/service/rds v1.113.1
	github.com/aws/aws-sdk-go-v2/service/redshift v1.61.4
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.31.5
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.93.2
	github.com/aws/aws-sdk-go-v2/service/s3vectors v1.6.1
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.228.2
	github.com/aws/aws-sdk-go-v2/service/savingsplans v1.31.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.0
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.67.2
	github.com/aws/aws-sdk-go-v2/service/servicequotas v1.33.12
	github.com/aws/aws-sdk-go-v2/service/sfn v1.40.5
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.10
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.20
	github.com/aws/aws-sdk-go-v2/service/ssm v1.67.7
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.5
	github.com/aws/aws-sdk-go-v2/service/transcribe v1.53.10
	github.com/aws/aws-sdk-go-v2/service/transfer v1.68.4
	github.com/aws/aws-sdk-go-v2/service/trustedadvisor v1.13.17
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.70.4
	github.com/aws/aws-sdk-go-v2/service/xray v1.36.16
	github.com/aws/smithy-go v1.24.0
	github.com/charmbracelet/x/ansi v0.11.3
	github.com/creack/pty v1.1.24
	golang.org/x/sync v0.19.0
	golang.org/x/term v0.38.0
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.5 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.12 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20251116181749-377898bcce38 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.6.1 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/sys v0.39.0 // indirect
)
