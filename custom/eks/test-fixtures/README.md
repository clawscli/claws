# EKS Test Fixtures

CloudFormation template for creating a minimal EKS cluster for integration testing with claws.

## Resources Created

- **VPC** (10.0.0.0/16)
- **2 Public Subnets** (10.0.1.0/24, 10.0.2.0/24) - for NAT Gateway
- **2 Private Subnets** (10.0.11.0/24, 10.0.12.0/24) - for worker nodes and Fargate
- **Internet Gateway**
- **NAT Gateway** (single instance for cost optimization) + Elastic IP
- **Route Tables** (Public + Private)
- **Security Groups** (Cluster SG + Node SG with proper ingress/egress rules)
- **EKS Cluster** (v1.31 with API_AND_CONFIG_MAP authentication mode)
- **Node Group** (1x t3.small instance in private subnets with Launch Template)
- **Fargate Profile** (fargate-test namespace in private subnets)
- **Addon** (vpc-cni)
- **Access Entry** (for testing)

All 5 EKS resource types supported by claws are included:
- clusters
- node-groups
- fargate-profiles
- addons
- access-entries

## Cost Estimate

- EKS Cluster: $0.10/hour ($73/month)
- t3.small instance: ~$0.021/hour (~$15/month)
- NAT Gateway: $0.045/hour (~$33/month)
- NAT Gateway data transfer: ~$0.045/GB (~$5-10/month estimated)
- **Total: ~$0.17/hour** (~$125/month if running 24/7)

**Recommended:** Deploy only when testing, then clean up immediately.
**Example:** 3 hours/day × 20 days = **~$10/month**

**Note:** Single NAT Gateway configuration (cost-optimized). For high availability, use 2 NAT Gateways (+$33/month).

## Prerequisites

- AWS CLI configured (`aws configure`)
- IAM permissions for:
  - CloudFormation
  - EKS
  - EC2 (VPC, Subnets, Internet Gateway)
  - IAM (Roles, Policies)

## Quick Start

### Using Task (Recommended)

```bash
# Deploy test stack
task eks-test-up

# Use claws to test
AWS_REGION=us-east-1 claws

# Clean up when done (auto-retries on DELETE_FAILED)
task eks-test-down
```

### Using Scripts Directly

```bash
cd custom/eks/test-fixtures/cloudformation

# Deploy (takes ~15-20 minutes)
./deploy.sh

# Configure kubectl (optional)
aws eks update-kubeconfig --name claws-test-cluster --region us-east-1

# Verify with kubectl (optional)
kubectl get nodes

# Test with claws
AWS_REGION=us-east-1 claws

# Clean up (takes ~10-15 minutes)
./cleanup.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STACK_NAME` | claws-eks-test | CloudFormation stack name |
| `CLUSTER_NAME` | claws-test-cluster | EKS cluster name |
| `AWS_REGION` | us-east-1 | AWS region |
| `CI_MODE` | false | Skip interactive prompts (cleanup.sh) |
| `MAX_RETRY` | 2 | Max deletion retry attempts (cleanup.sh) |

Example with custom values:

```bash
STACK_NAME=my-test \
CLUSTER_NAME=my-cluster \
AWS_REGION=ap-northeast-1 \
./deploy.sh
```

## Testing with claws

After deployment:

```bash
# Set region
export AWS_REGION=us-east-1

# Run claws
claws

# Navigate to EKS resources:
# - EKS > clusters
# - EKS > node-groups
# - EKS > fargate-profiles
# - EKS > addons
# - EKS > access-entries
```

## Troubleshooting

### Stack already exists

If you see "Stack already exists", the script will update the existing stack instead of creating a new one.

### Deployment timeout

EKS cluster creation typically takes 15-20 minutes. If it times out, check the CloudFormation console for detailed error messages:

```bash
aws cloudformation describe-stack-events \
  --stack-name claws-eks-test \
  --region us-east-1 \
  --max-items 20
```

### Cleanup fails

**NEW:** cleanup.sh now auto-retries on DELETE_FAILED and fixes auth mode issues automatically.

If cleanup still fails after retries:

```bash
# Check failure reason
aws cloudformation describe-stack-events \
  --stack-name claws-eks-test \
  --region us-east-1 \
  --query 'StackEvents[?ResourceStatus==`DELETE_FAILED`]' \
  --max-items 10

# For LoadBalancers created by k8s services:
aws elbv2 describe-load-balancers --region us-east-1 \
  --query 'LoadBalancers[?VpcId==`<VPC_ID>`]'

# Delete manually, then retry
CI_MODE=true ./cleanup.sh  # Skip confirmation prompt
```

## Security

### Security Groups

The template creates secure security groups with minimal required access:

**Cluster Security Group:**
- Allows communication from worker nodes on port 443

**Node Security Group:**
- Allows inbound from control plane (ports 1025-65535, 443)
- Allows all traffic between nodes (pod-to-pod communication)
- Allows all outbound traffic (for ECR, AWS APIs, package updates)
- **Blocks all other inbound traffic from the internet**

### Network Architecture

**Production-grade private subnet architecture:**

```
VPC (10.0.0.0/16)
├── Public Subnets (10.0.1.0/24, 10.0.2.0/24)
│   └── NAT Gateway + Elastic IP
└── Private Subnets (10.0.11.0/24, 10.0.12.0/24)
    ├── EKS Worker Nodes (no public IPs)
    └── Fargate Pods (no public IPs)
```

**Traffic Flow:**
- Worker nodes/Fargate → NAT Gateway → Internet Gateway → AWS Services (ECR, EKS API)
- No direct internet access for worker nodes
- Security groups enforce strict ingress/egress rules

**Why this architecture?**
- **Fargate requirement**: Fargate profiles only support private subnets
- **Access Entry requirement**: Cluster must have API or API_AND_CONFIG_MAP authentication mode
- **Security**: Worker nodes have no public IPs, cannot be directly accessed from internet
- **AWS best practices**: Follows EKS production deployment patterns

### Additional Security Features

- **IMDSv2 enforced** on worker nodes (metadata token required)
- **No SSH access** configured (can be added if needed)
- **IAM roles** follow AWS best practices

## Notes

- **Do not commit AWS credentials** - Use IAM roles or AWS CLI profiles
- **Remember to clean up** - NAT Gateway costs add up quickly ($0.045/hour)
- **Test environment** - Single NAT Gateway (no HA), minimal node count
- **Production-ready architecture** - Private subnets, proper security groups, follows AWS best practices
- **Fargate compatible** - All 5 EKS resource types can be tested including Fargate profiles
- **Access Entry enabled** - Cluster configured with API_AND_CONFIG_MAP authentication mode
