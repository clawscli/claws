# claws

A terminal UI for AWS resource management 👮

[![CI](https://github.com/clawscli/claws/actions/workflows/ci.yml/badge.svg)](https://github.com/clawscli/claws/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/clawscli/claws)](https://github.com/clawscli/claws/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/clawscli/claws)](https://goreportcard.com/report/github.com/clawscli/claws)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

![claws demo](demo.gif)

## Supported Platforms

| OS | Architecture |
|-----|-------------|
| macOS | Intel, Apple Silicon |
| Linux | x86_64, ARM64 |
| Windows | x86_64 |

## Features

- **Interactive TUI** - Navigate AWS resources with vim-style keybindings
- **Mouse support** - Click, scroll, hover for navigation
- **Multi-service support** - EC2, S3, IAM, RDS, Lambda, ECS, and 65+ more services (163 resources total)
- **Resource actions** - Start/stop instances, delete resources, tail logs
- **Cross-resource navigation** - Jump from VPC to subnets, from Lambda to CloudWatch Logs
- **Profile & region switching** - Switch AWS profiles (`P`) and regions (`R`) on the fly
- **Multi-profile selection** - Select multiple profiles with `P`, parallel fetch across accounts
- **Multi-region selection** - Select multiple regions with `R`, parallel fetch with aggregated results
- **Command mode** - Quick navigation with `:ec2/instances` syntax
- **Tag search** - Browse all tagged resources across regions with `:tags` command
- **Filtering** - Fuzzy search with `/`, tag filtering with `:tag Env=prod`
- **Column sorting** - Sort by any column with `:sort <col>` command
- **Resource comparison** - Side-by-side diff view with `m` to mark, `d` to compare
- **Pagination** - Handle large datasets with `N` key for next page

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap clawscli/tap
brew install --cask claws
```

### Install Script (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/clawscli/claws/main/install.sh | sh
```

Options:
```bash
# Install specific version
curl -fsSL https://raw.githubusercontent.com/clawscli/claws/main/install.sh | VERSION=v0.1.6 sh

# Install to custom directory
curl -fsSL https://raw.githubusercontent.com/clawscli/claws/main/install.sh | INSTALL_DIR=/usr/local/bin sh
```

> **Security Note:** Piping scripts directly to `sh` is convenient but bypasses inspection.
> For increased security, download and review the script first:
> ```bash
> curl -fsSL https://raw.githubusercontent.com/clawscli/claws/main/install.sh -o install.sh
> less install.sh  # Review the script
> sh install.sh
> ```

### Download Binary

Download from [GitHub Releases](https://github.com/clawscli/claws/releases/latest):

```bash
# macOS (Apple Silicon)
curl -Lo claws.tar.gz https://github.com/clawscli/claws/releases/latest/download/claws-darwin-arm64.tar.gz
tar xzf claws.tar.gz && mv claws /usr/local/bin/

# macOS (Intel)
curl -Lo claws.tar.gz https://github.com/clawscli/claws/releases/latest/download/claws-darwin-amd64.tar.gz
tar xzf claws.tar.gz && mv claws /usr/local/bin/

# Linux (x86_64)
curl -Lo claws.tar.gz https://github.com/clawscli/claws/releases/latest/download/claws-linux-amd64.tar.gz
tar xzf claws.tar.gz && sudo mv claws /usr/local/bin/

# Linux (ARM64)
curl -Lo claws.tar.gz https://github.com/clawscli/claws/releases/latest/download/claws-linux-arm64.tar.gz
tar xzf claws.tar.gz && sudo mv claws /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/clawscli/claws/releases/latest/download/claws-windows-amd64.zip -OutFile claws.zip
Expand-Archive claws.zip -DestinationPath .
# Add to PATH or move to desired location
```

### Go Install

```bash
go install github.com/clawscli/claws/cmd/claws@latest
```

### From Source

```bash
git clone https://github.com/clawscli/claws.git
cd claws
go build -o claws ./cmd/claws
```

## Quick Start

```bash
# Run claws (uses default AWS credentials)
claws

# Or with specific profile
claws -p myprofile

# Or with specific region
claws -r us-west-2

# Use environment credentials only (ignore ~/.aws config)
# Useful for EC2 instance profiles, ECS task roles, Lambda, etc.
claws -e

# Read-only mode (disables destructive actions)
claws --read-only
# or
CLAWS_READ_ONLY=1 claws

# Enable debug logging to file
claws -l debug.log
```

## Key Bindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `h` / `l` | Navigate within category (service list) |
| `Enter` / `d` | View resource details |
| `:` | Command mode (e.g., `:ec2/instances`) |
| `:` + `Enter` | Go to dashboard (home) |
| `~` | Go to dashboard (from service browser) |
| `:services` | Go to service browser |
| `:sort <col>` | Sort by column (ascending) |
| `:sort desc <col>` | Sort by column (descending) |
| `:tag <filter>` | Filter by tag (e.g., `:tag Env=prod`) |
| `:tags` | Browse all tagged resources |
| `/` | Filter mode (fuzzy search) |
| `Tab` | Next resource type |
| `1-9` | Switch to resource type by number |
| `a` | Open actions menu |
| `m` | Mark resource for comparison |
| `d` | Describe (or diff if marked) |
| `c` | Clear filter and mark |
| `N` | Load next page (pagination) |
| `M` | Toggle inline metrics (EC2, RDS, Lambda) |
| `Ctrl+r` | Refresh (including metrics) |
| `R` | Select AWS region(s) (multi-select supported) |
| `P` | Select AWS profile(s) (multi-select supported) |
| `?` | Show help |
| `Esc` | Go back |
| `q` / `Ctrl+c` | Quit |

### Mouse Support

| Action | Effect |
|--------|--------|
| Hover | Highlight item under cursor |
| Click | Select item / navigate |
| Scroll wheel | Scroll through lists |
| Click on tabs | Switch resource type |
| Back button | Navigate back (same as Esc) |

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

### Region Selector (`R` key)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Space` | Toggle region selection |
| `a` | Select all regions |
| `n` | Deselect all regions |
| `/` | Filter regions |
| `Enter` | Apply selection |
| `Esc` | Cancel |

Selected regions are queried in parallel; resources display with Region column.

### Profile Selector (`P` key)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Space` | Toggle profile selection |
| `l` | SSO login for selected profile |
| `L` | Console login for selected profile (`:login`) |
| `/` | Filter profiles |
| `Enter` | Apply selection |
| `Esc` | Cancel |

Selected profiles are queried in parallel; resources display with Profile and Account columns.

### Commands

| Command | Action |
|---------|--------|
| `:q` / `:quit` | Quit |
| `:login [name]` | AWS console login (default: `claws-login` profile) |
| `:ec2/instances` | Navigate to EC2 instances |
| `:sort <col>` | Sort by column |
| `:tag <filter>` | Filter by tag |
| `:diff <name>` | Compare current row with named resource |
| `:diff <n1> <n2>` | Compare two named resources |

**Login Details:**
- `:login` runs `aws login --remote` using `claws-login` profile
- `:login myprofile` uses the specified profile name instead
- For SSO profiles, use `P` to open profile selector, then `l` for SSO login

## Supported Services (69 services, 163 resources)

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
| CloudWatch | Alarms, Log Groups, Log Streams |
| CloudTrail | Trails, Events |
| AWS Config | Rules |
| AWS Health | Events |
| X-Ray | Groups |
| Service Quotas | Services, Quotas |
| CodeBuild | Projects, Builds |
| CodePipeline | Pipelines, Executions |
| AWS Backup | Plans, Vaults, Selections, Protected Resources, Backup Jobs, Copy Jobs, Restore Jobs, Recovery Points |
| Organizations | Accounts, OUs, Policies, Roots |
| License Manager | Configurations, Licenses, Grants |

### Cost Management
| Service | Resources |
|---------|-----------|
| RI/SP | Reserved Instances, Savings Plans |
| Cost Explorer | Costs, Anomalies, Monitors |
| Compute Optimizer | Summary, Recommendations |
| Trusted Advisor | Recommendations |
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
| `co` | Compute Optimizer |
| `ta` | Trusted Advisor |
| `ri` | Reserved Instances |
| `sp` | Savings Plans |
| `odcr` | Capacity Reservations |
| `tgw` | Transit Gateways |
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

### Configuration File

Optional settings can be stored in `~/.config/claws/config.yaml`:

```yaml
timeouts:
  aws_init: 10s           # AWS initialization timeout (default: 5s)
  multi_region_fetch: 60s # Multi-region parallel fetch timeout (default: 30s)
  tag_search: 45s         # Tag search timeout (default: 30s)
  metrics_load: 30s       # CloudWatch metrics load timeout (default: 30s)

concurrency:
  max_fetches: 100        # Max concurrent API fetches (default: 50)

cloudwatch:
  window: 15m             # Metrics data window period (default: 15m)

persistence:
  enabled: true           # Save region/profile on change (default: false)

startup:                  # Applied on launch if present
  profile: production
  regions:
    - us-east-1
    - us-west-2
```

The config file is **not created automatically**. Create it manually if needed.

CLI flags (`-p`, `-r`, `--persist`, `--no-persist`) override config file settings.

For required IAM permissions, see [docs/iam-permissions.md](docs/iam-permissions.md).

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

### Releasing

Releases are automated via GoReleaser. Tag format controls which platforms are built:

| Tag | Platforms Built |
|-----|-----------------|
| `v0.1.0` | All (linux/darwin arm64+amd64, windows) |
| `v0.1.0-rc1` | ARM64 only (linux/darwin) |
| `v0.1.0-rc1-amd64` | ARM64 + AMD64 (linux/darwin) |
| `v0.1.0-rc1-win` | ARM64 + Windows |
| `v0.1.0-rc1-all` | All platforms |

Prereleases (tags containing `-`) skip Homebrew publishing.

## Tech Stack

- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)
- **AWS SDK**: [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
