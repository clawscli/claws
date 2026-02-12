# 지원 서비스

claws는 **70개 서비스**와 **175개 리소스**를 지원합니다.

## 컴퓨팅

| Service | Resources |
|---------|-----------|
| EC2 | Instances, Volumes, Security Groups, Elastic IPs, Key Pairs, AMIs, Snapshots, Launch Templates, Capacity Reservations |
| Lambda | Functions |
| ECS | Clusters, Services, Tasks, Task Definitions |
| Auto Scaling | Groups, Activities |
| App Runner | Services, Operations |
| Batch | Job Queues, Compute Environments, Jobs, Job Definitions |
| EMR | Clusters, Steps |

## 스토리지 및 데이터베이스

| Service | Resources |
|---------|-----------|
| S3 | Buckets |
| S3 Vectors | Buckets, Indexes |
| DynamoDB | Tables |
| RDS | Instances, Snapshots |
| Redshift | Clusters, Snapshots |
| ElastiCache | Clusters |
| OpenSearch | Domains |

## 데이터 및 분석

| Service | Resources |
|---------|-----------|
| Glue | Databases, Tables, Crawlers, Jobs, Job Runs |
| Athena | Workgroups, Query Executions |
| Transcribe | Jobs |

## 컨테이너 및 ML

| Service | Resources |
|---------|-----------|
| ECR | Repositories, Images |
| EKS | Clusters, Node Groups, Fargate Profiles, Addons, Access Entries |
| Bedrock | Foundation Models, Guardrails, Inference Profiles |
| Bedrock Agent | Agents, Knowledge Bases, Data Sources, Prompts, Flows |
| Bedrock AgentCore | Runtimes, Endpoints, Versions |
| SageMaker | Endpoints, Notebooks, Training Jobs, Models |

## 네트워킹

| Service | Resources |
|---------|-----------|
| VPC | VPCs, Subnets, Route Tables, Internet Gateways, NAT Gateways, VPC Endpoints, Transit Gateways, TGW Attachments |
| Route 53 | Hosted Zones, Record Sets |
| API Gateway | REST APIs, HTTP APIs, Stages |
| AppSync | GraphQL APIs, Data Sources |
| ELB | Load Balancers, Target Groups, Targets |
| CloudFront | Distributions |
| Direct Connect | Connections, Virtual Interfaces |

## 보안 및 ID

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

## 통합

| Service | Resources |
|---------|-----------|
| SQS | Queues |
| SNS | Topics, Subscriptions |
| EventBridge | Event Buses, Rules |
| Step Functions | State Machines, Executions |
| Kinesis | Streams |
| Transfer Family | Servers, Users |
| DataSync | Tasks, Locations, Task Executions |

## 관리 및 모니터링

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

## 비용 관리

| Service | Resources |
|---------|-----------|
| RI/SP | Reserved Instances, Savings Plans |
| Cost Explorer | Costs, Anomalies, Monitors |
| Compute Optimizer | Summary, Recommendations |
| Trusted Advisor | Recommendations |
| Budgets | Budgets, Notifications |

## 게임 개발

| Service | Resources |
|---------|-----------|
| GameLift | Fleets, Builds, Scripts, Game Sessions, Game Session Queues, Matchmaking Configurations |

---

## 서비스 별칭

자주 사용하는 서비스의 빠른 단축키입니다:

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
| `eks` | EKS |
| `gl` | GameLift |
