# claws - AWS TUI Project Overview

## Purpose
k9s-inspired TUI for AWS services with cross-resource navigation, action support, and comprehensive AWS resource management.

## Tech Stack
- **Language**: Go 1.23+
- **TUI**: Bubbletea v1.3.10 + Lipgloss v1.1.0 + Bubbles v0.21.0
- **AWS SDK**: SDK for Go v2 v1.41.0
- **Actions**: Defined in Go code within each resource's actions.go

## Current Status (2025-12)
- **Phase**: Feature complete, maintenance mode
- **Files**: 552 Go files, ~74,000 lines
- **Services**: 65 services, 146 resources
- **Test Coverage**: internal/ ~60% avg, view/ 18.7%

## Architecture
All implementations in `custom/` directory. No code generation.

## Directory Structure
```
claws/
├── cmd/claws/           # Entry point
├── internal/
│   ├── app/             # Main TUI app
│   ├── aws/             # AWS client, helpers (Paginate, IsNotFound, etc.)
│   ├── action/          # Action framework
│   ├── config/          # App config (profile, region)
│   ├── dao/             # Data access interface + context filtering
│   ├── log/             # Structured logging (slog-based)
│   ├── render/          # Renderer + DetailBuilder + Navigation
│   ├── registry/        # Service registry + aliases + sub-resources
│   ├── ui/              # Theme system
│   └── view/            # Views (browser, detail, command, help, selectors)
├── custom/              # All 65 service implementations
│   ├── ec2/             # EC2 (13 resources)
│   ├── iam/             # IAM (5 resources)
│   ├── glue/            # Glue (5 resources)
│   ├── bedrock/         # Bedrock (3 resources)
│   ├── bedrockagent/    # Bedrock Agent (6 resources)
│   └── ...              # 60+ more services

```

## Key Helpers (internal/aws/)
- `appaws.NewConfig(ctx)` - Centralized AWS config loading
- `appaws.Paginate(ctx, fetchFn)` - Generic batch pagination
- `appaws.PaginateIter(ctx, fetchFn)` - Streaming iterator pagination
- `appaws.IsNotFound(err)` - Check for "not found" errors
- `appaws.IsAccessDenied(err)` - Check for "access denied" errors
- `appaws.IsThrottling(err)` - Check for rate limiting
- `appaws.WithAPITimeout(ctx)` - Add 30s timeout to context
- `appaws.Str/Int32/Int64/Time(ptr)` - Safe pointer dereferencing

## Key Helpers (internal/render/)
- `render.FormatAge(time)` - Human-readable age formatting
- `render.NewDetailBuilder()` - Consistent detail view building
- `render.Navigation{}` - Cross-resource navigation definition

## Key Helpers (internal/dao/)
- `dao.WithFilter(ctx, field, value)` - Context-based filtering
- `dao.GetFilterFromContext(ctx, field)` - Retrieve filter value

## Service Categories
- **Compute**: ec2, lambda, ecs, autoscaling, apprunner, batch, emr
- **Storage/DB**: s3, s3vectors, dynamodb, rds, redshift, elasticache, opensearch
- **Data/Analytics**: glue, athena, transcribe
- **Containers/ML**: ecr, bedrock, bedrock-agent, bedrock-agentcore, sagemaker
- **Networking**: vpc, route53, apigateway, appsync, elbv2, cloudfront, directconnect, network-firewall
- **Security/Identity**: iam, kms, acm, secretsmanager, ssm, cognito, guardduty, wafv2, inspector2, securityhub, fms, accessanalyzer, detective, macie
- **Integration**: sqs, sns, eventbridge, sfn, kinesis, transfer, datasync
- **Management/Monitoring**: cloudformation, cloudwatch, cloudtrail, config, health, xray, service-quotas, codebuild, codepipeline, backup, organizations, license-manager
- **Cost**: costexplorer, budgets

## Key Features
- Profile switching (`P` key)
- Region switching (`R` key)
- Cross-resource navigation
- Column sorting (`:sort <col>`)
- Pagination for large datasets (`N` key)
- Go-based action definitions
- Read-only mode (`--read-only`)
- Debug logging (`-l/--log-file`)

## Commands
```bash
task build    # Build binary
task run      # Run app
task test     # Run tests
task lint     # Run linters
```

## Known Issues
- Mouse tracking disabled (causes ESC key buffering issues)
