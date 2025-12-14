# claws - AWS TUI Project

## Project Configuration
SNAP_PROJECT_NAME: claws

## Overview
AWS TUI for browsing and managing AWS resources 👮

## Current Status
- **Phase**: Feature complete, maintenance mode
- **Blockers**: None
- **Last Updated**: 2025-12-14
- **Code**: 550+ Go files, ~74,000 lines
- **Test Coverage**: internal/ ~60% avg, view/ 18.7% (TUI code)
- **Services**: 66 services, 146 resources
- **TODOs**: 0 (codebase clean)

## Known Issues
- Mouse tracking disabled: `tea.WithMouseCellMotion()` causes ESC key buffering issues. Keep disabled in `cmd/claws/main.go`.

## Decided Tech Stack
- **TUI**: Bubbletea v1.3.10 + Lipgloss v1.1.0 + Bubbles v0.21.0
- **AWS**: SDK for Go v2 v1.41.0

## Architecture
- All DAO/Renderer/Action implementations are in `custom/`
- Each resource has: `dao.go`, `render.go`, `register.go`, and optionally `actions.go`

## Implemented Services (66)
accessanalyzer, acm, apigateway, apprunner, appsync, athena, autoscaling, backup, batch, bedrock, bedrock-agent, bedrock-agentcore, budgets, cloudformation, cloudfront, cloudtrail, cloudwatch, codebuild, codepipeline, cognito, config, costexplorer, datasync, detective, directconnect, dynamodb, ec2, ecr, ecs, elasticache, elbv2, emr, eventbridge, fms, glue, guardduty, health, iam, inspector2, kinesis, kms, lambda, license-manager, macie, network-firewall, opensearch, organizations, rds, redshift, route53, s3, s3vectors, sagemaker, secretsmanager, securityhub, service-quotas, sfn, sns, sqs, ssm, transcribe, transfer, wafv2, xray

## Service Categories
- **Compute**: ec2, lambda, ecs, autoscaling, apprunner, batch, emr
- **Storage & Database**: s3, s3vectors, dynamodb, rds, redshift, elasticache, opensearch
- **Data & Analytics**: glue, athena, transcribe
- **Containers & ML**: ecr, bedrock, bedrock-agent, bedrock-agentcore, sagemaker
- **Networking**: vpc, vpc-endpoints, transit-gateways, route53, apigateway, appsync, elbv2, cloudfront, directconnect, network-firewall
- **Security & Identity**: iam, kms, acm, secretsmanager, ssm, cognito, guardduty, wafv2, inspector2, securityhub, fms, accessanalyzer, detective, macie
- **Integration**: sqs, sns, eventbridge, sfn, kinesis, transfer, datasync
- **Management & Monitoring**: cloudformation, cloudwatch, cloudtrail, config, health, xray, service-quotas, codebuild, codepipeline, backup, organizations, license-manager
- **Cost Management**: costexplorer, budgets

## Key Bindings
| Key | Action |
|-----|--------|
| `j/k` | Navigate |
| `Enter/d` | Describe |
| `:` | Command mode |
| `:` + `Enter` | Go to service list (home) |
| `:sort <col>` | Sort by column (ascending) |
| `:sort desc <col>` | Sort by column (descending) |
| `:sort` | Clear sorting |
| `:tag <filter>` | Filter by tag (e.g., `:tag Env=prod`) |
| `:tag` | Clear tag filter |
| `:tags` | Browse all tagged resources |
| `:tags <filter>` | Browse with tag filter |
| `/` | Filter (fuzzy search) |
| `Tab` | Next resource type |
| `1-9` | Switch to resource type |
| `a` | Actions menu |
| `c` | Clear filter |
| `N` | Load next page (pagination) |
| `Ctrl+r` | Refresh |
| `R` | Switch AWS region |
| `P` | Switch AWS profile |
| `?` | Help |
| `esc` | Back |
| `Ctrl+c` | Quit |

## Service Aliases
| Alias | Target |
|-------|--------|
| `cfn` | cloudformation |
| `cf` | cloudfront |
| `sg` | ec2/security-groups |
| `asg` | autoscaling |
| `logs` | cloudwatch/log-groups |
| `cw` | cloudwatch |
| `ddb` | dynamodb |
| `sm` | secretsmanager |
| `r53` | route53 |
| `agentcore` | bedrock-agentcore |
| `kb` | bedrock-agent/knowledge-bases |
| `agent` | bedrock-agent/agents |
| `models` | bedrock/foundation-models |
| `guardrail` | bedrock/guardrails |
| `eb` | eventbridge |
| `sfn` | sfn (Step Functions) |
| `sq`, `quotas` | service-quotas |
| `apigw`, `api` | apigateway |
| `elb`, `alb`, `nlb` | elbv2 |
| `redis`, `cache` | elasticache |
| `es`, `elasticsearch` | opensearch |
| `cdn`, `dist` | cloudfront |
| `gd` | guardduty |
| `build`, `cb` | codebuild |
| `pipeline`, `cp` | codepipeline |
| `waf` | wafv2 |
| `ce`, `cost-explorer` | costexplorer |

## Sub-Resources (Navigation Only)
- `cloudformation/events` - from stacks (e key)
- `cloudformation/resources` - from stacks (r key)
- `cloudformation/outputs` - from stacks (o key)
- `cloudwatch/log-streams` - from log-groups (s key)
- `service-quotas/quotas` - from services (q key)
- `route53/record-sets` - from hosted-zones (r key)
- `apigateway/stages` - from rest-apis (s key)
- `apigateway/stages-v2` - from http-apis (s key)
- `elbv2/targets` - from target-groups (t key)
- `s3vectors/indexes` - from buckets (i key)
- `guardduty/findings` - from detectors (f key)
- `cognito/users` - from userpools (u key)
- `codepipeline/executions` - from pipelines (e key)
- `sfn/executions` - from state-machines (e key)
- `codebuild/builds` - from projects (b key)
- `backup/jobs` - from plans (o key)
- `ecr/images` - from repositories (i key)
- `autoscaling/activities` - from groups (a key)
- `bedrock-agent/data-sources` - from knowledge-bases (s key)
- `bedrock-agentcore/endpoints` - from runtimes (e key)
- `bedrock-agentcore/versions` - from runtimes (v key)
- `transfer/users` - from servers (u key)
- `accessanalyzer/findings` - from analyzers (f key)
- `detective/investigations` - from graphs (i key)
- `datasync/task-executions` - from tasks (e key)
- `batch/jobs` - from job-queues (j key)
- `emr/steps` - from clusters (s key)
- `organizations/ous` - from roots (o key)
- `license-manager/grants` - from licenses (g key)
- `appsync/data-sources` - from graphql-apis (d key)
- `redshift/snapshots` - from clusters (s key)

## CLI Flags
```bash
claws [options]

Options:
  -p, --profile <name>   AWS profile to use
  -r, --region <region>  AWS region to use
  -ro, --read-only       Run in read-only mode
  -l, --log-file <path>  Enable debug logging to file
  -h, --help             Show help message
```

## Commands (Taskfile)
```bash
task build           # Build binary
task run             # Run app
task test            # Run unit tests
task test-cover      # Run tests with coverage
task lint            # Run linters
task fmt             # Format code
```

## Adding a New Resource (How-To)

### 1. Create custom DAO/Renderer
Create files in `custom/<service>/<resource>/`:
- `dao.go` - DAO + Resource type
- `render.go` - Renderer (columns, detail, summary, navigation)
- `register.go` - Registry registration

### 2. Add import to main.go
```go
_ "github.com/clawscli/claws/custom/<service>/<resource>"
```

### 3. Add actions (optional)
Create `actions.go` in the same directory with:
```go
func init() {
    action.Global.Register("service", "resource", []action.Action{...})
    action.RegisterExecutor("service", "resource", executeAction)
}
```

## Recent Changes (2025-12-14)
- **License**: Changed to Apache 2.0
- **CI/CD**: Added GitHub Actions (CI + GoReleaser for releases)
- **Module path**: Changed to `github.com/clawscli/claws`
- **Dependencies**: Updated charmbracelet libs, golang.org/x/text v0.32.0

## Critical Implementation Notes
- `matchesFieldFilter` in `resource_browser_filter.go`: Must compare full ARN BEFORE extracting resource name
- Navigations method signature: `Navigations(resource dao.Resource) []render.Navigation`
- DAO filter: Use `dao.GetFilterFromContext(ctx, "FieldName")` in List()
- Pointer dereferencing: Always use `appaws.Str()`, `appaws.Int32()`, `appaws.Int64()`, `appaws.Time()` helpers
- Pagination: Use `appaws.Paginate` for batch collection, `appaws.PaginateIter` for per-item processing
- **PaginatedDAO**: Implement `ListPage(ctx, pageSize, pageToken)` for large datasets. CloudTrail requires same StartTime/EndTime for pagination tokens.
