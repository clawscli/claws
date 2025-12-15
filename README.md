# claws

A terminal UI for AWS resource management 👮

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

![claws demo](demo.gif)

## Features

- **Interactive TUI** - Navigate AWS resources with vim-style keybindings
- **Multi-service support** - EC2, S3, IAM, RDS, Lambda, ECS, and 60+ more services
- **Resource actions** - Start/stop instances, delete resources, tail logs
- **Cross-resource navigation** - Jump from VPC to subnets, from Lambda to CloudWatch Logs
- **Profile & region switching** - Switch AWS profiles (`P`) and regions (`R`) on the fly
- **Command mode** - Quick navigation with `:ec2/instances` syntax
- **Filtering** - Fuzzy search with `/`, tag filtering with `:tag Env=prod`
- **Column sorting** - Sort by any column with `:sort <col>` command
- **Pagination** - Handle large datasets with `N` key for next page

## Installation

### From Source

```bash
git clone https://github.com/clawscli/claws.git
cd claws
go build -o claws ./cmd/claws
```

### Using Task

```bash
task build
```

## Quick Start

```bash
# Run claws (uses default AWS credentials)
./claws

# Or with specific profile
./claws -p myprofile

# Or with specific region
./claws -r us-west-2

# Read-only mode (disables destructive actions)
./claws --read-only

# Enable debug logging to file
./claws -l debug.log
```

## Key Bindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` / `d` | View resource details |
| `:` | Command mode (e.g., `:ec2/instances`) |
| `:` + `Enter` | Go to service list (home) |
| `:sort <col>` | Sort by column (ascending) |
| `:sort desc <col>` | Sort by column (descending) |
| `:tag <filter>` | Filter by tag (e.g., `:tag Env=prod`) |
| `:tags` | Browse all tagged resources |
| `/` | Filter mode (fuzzy search) |
| `Tab` | Next resource type |
| `1-9` | Switch to resource type by number |
| `a` | Open actions menu |
| `c` | Clear filter |
| `N` | Load next page (pagination) |
| `Ctrl+r` | Refresh |
| `R` | Switch AWS region |
| `P` | Switch AWS profile |
| `?` | Show help |
| `Esc` | Go back |
| `Ctrl+c` | Quit |

### Navigation Shortcuts (Context-dependent)

| Key | Action |
|-----|--------|
| `v` | View VPC / Versions |
| `s` | View Subnets / Streams / Stages |
| `g` | View Security Groups |
| `r` | View Route Tables / Roles / Resources |
| `e` | View Events / Executions / Endpoints |
| `l` | View CloudWatch Logs |
| `o` | View Outputs / Operations |
| `i` | View Images / Indexes |

## Supported Services (67 services, 151 resources)

### Compute
| Service | Resources |
|---------|-----------|
| EC2 | Instances, Volumes, Security Groups, Elastic IPs, Key Pairs, AMIs, Snapshots, Launch Templates, Capacity Reservations |
| Lambda | Functions |
| ECS | Clusters, Services, Tasks |
| Auto Scaling | Groups, Activities |
| App Runner | Services, Operations |
| Batch | Job Queues, Compute Environments, Jobs, Job Definitions |
| EMR | Clusters, Steps |

### Storage & Database
| Service | Resources |
|---------|-----------|
| S3 | Buckets |
| S3 Vectors | Buckets, Indexes |
| DynamoDB | Tables |
| RDS | Instances, Snapshots |
| Redshift | Clusters, Snapshots |
| ElastiCache | Clusters |
| OpenSearch | Domains |

### Data & Analytics
| Service | Resources |
|---------|-----------|
| Glue | Databases, Tables, Crawlers, Jobs, Job Runs |
| Athena | Workgroups, Query Executions |
| Transcribe | Jobs |

### Containers & ML
| Service | Resources |
|---------|-----------|
| ECR | Repositories, Images |
| Bedrock | Foundation Models, Guardrails, Inference Profiles |
| Bedrock Agent | Agents, Knowledge Bases, Data Sources, Prompts, Flows |
| Bedrock AgentCore | Runtimes, Endpoints, Versions |
| SageMaker | Endpoints, Notebooks, Training Jobs, Models |

### Networking
| Service | Resources |
|---------|-----------|
| VPC | VPCs, Subnets, Route Tables, Internet Gateways, NAT Gateways, VPC Endpoints, Transit Gateways, TGW Attachments |
| Route 53 | Hosted Zones, Record Sets |
| API Gateway | REST APIs, HTTP APIs, Stages |
| AppSync | GraphQL APIs, Data Sources |
| ELB | Load Balancers, Target Groups, Targets |
| CloudFront | Distributions |
| Direct Connect | Connections, Virtual Interfaces |

### Security & Identity
| Service | Resources |
|---------|-----------|
| IAM | Users, Roles, Policies, Groups, Instance Profiles |
| KMS | Keys |
| ACM | Certificates |
| Secrets Manager | Secrets |
| SSM | Parameters |
| Cognito | User Pools, Users |
| GuardDuty | Detectors, Findings |
| WAF | Web ACLs |
| Inspector | Findings |
| Security Hub | Findings |
| Firewall Manager | Policies |
| Network Firewall | Firewalls, Firewall Policies, Rule Groups |
| IAM Access Analyzer | Analyzers, Findings |
| Detective | Graphs, Investigations |
| Macie | Classification Jobs, Findings, Buckets |

### Integration
| Service | Resources |
|---------|-----------|
| SQS | Queues |
| SNS | Topics, Subscriptions |
| EventBridge | Event Buses, Rules |
| Step Functions | State Machines, Executions |
| Kinesis | Streams |
| Transfer Family | Servers, Users |
| DataSync | Tasks, Locations, Task Executions |

### Management & Monitoring
| Service | Resources |
|---------|-----------|
| CloudFormation | Stacks, Events, Resources, Outputs |
| CloudWatch | Log Groups, Log Streams |
| CloudTrail | Trails, Events |
| AWS Config | Rules |
| AWS Health | Events |
| X-Ray | Groups |
| Service Quotas | Services, Quotas |
| CodeBuild | Projects, Builds |
| CodePipeline | Pipelines, Executions |
| AWS Backup | Plans, Jobs |
| Organizations | Accounts, OUs, Policies, Roots |
| License Manager | Configurations, Licenses, Grants |

### Cost Management
| Service | Resources |
|---------|-----------|
| RI/SP | Reserved Instances, Savings Plans |
| Cost Explorer | Costs |
| Budgets | Budgets, Notifications |

## Service Aliases

Quick shortcuts for common services:

| Alias | Service |
|-------|---------|
| `cfn`, `cf` | CloudFormation |
| `sg` | EC2 Security Groups |
| `asg` | Auto Scaling |
| `cw` | CloudWatch |
| `logs` | CloudWatch Log Groups |
| `ddb` | DynamoDB |
| `sm` | Secrets Manager |
| `r53` | Route 53 |
| `eb` | EventBridge |
| `sfn` | Step Functions |
| `sq`, `quotas` | Service Quotas |
| `apigw`, `api` | API Gateway |
| `elb`, `alb`, `nlb` | Elastic Load Balancing |
| `redis`, `cache` | ElastiCache |
| `es`, `elasticsearch` | OpenSearch |
| `cdn`, `dist` | CloudFront |
| `gd` | GuardDuty |
| `build`, `cb` | CodeBuild |
| `pipeline`, `cp` | CodePipeline |
| `waf` | WAF |
| `ce`, `cost-explorer` | Cost Explorer |
| `ri` | Reserved Instances |
| `sp` | Savings Plans |
| `odcr` | Capacity Reservations |
| `agentcore` | Bedrock AgentCore |
| `kb` | Bedrock Agent Knowledge Bases |
| `agent` | Bedrock Agent Agents |
| `models` | Bedrock Foundation Models |
| `guardrail` | Bedrock Guardrails |

## Configuration

claws uses your standard AWS configuration:

- `~/.aws/credentials` - AWS credentials
- `~/.aws/config` - AWS configuration (region, profile)
- Environment variables: `AWS_PROFILE`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, etc.

Configuration is stored in `~/.config/claws/config.yaml` for profile preferences.

## Architecture

claws uses a simple architecture with custom implementations for each service:

```
claws/
├── cmd/claws/           # Entry point
├── internal/
│   ├── app/             # Main TUI application
│   ├── aws/             # AWS client management + helpers
│   ├── action/          # Action framework
│   ├── dao/             # Data Access Object interface
│   ├── log/             # Structured logging (slog-based)
│   ├── registry/        # Service registry + aliases
│   ├── render/          # Renderer interface
│   ├── ui/              # Theme system
│   └── view/            # View components
└── custom/              # Service implementations (DAO + Renderer + Actions)
```

See [docs/architecture.md](docs/architecture.md) for details.

## Development

### Prerequisites

- Go 1.25+
- [Task](https://taskfile.dev/) (optional, for task runner)

### Commands

```bash
task build          # Build binary
task run            # Run the application
task test           # Run tests
task test-cover     # Run tests with coverage
task lint           # Run linters
task fmt            # Format code
task clean          # Clean build artifacts
```

### Adding New Resources

See [docs/adding-resources.md](docs/adding-resources.md) for a guide on adding new AWS resources.

## Tech Stack

- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)
- **AWS SDK**: [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
