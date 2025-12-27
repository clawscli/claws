#!/bin/bash
# LocalStack Demo Setup Script
# Creates demo resources for claws VHS recording
#
# Safety: Only runs against localhost:4566

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() { echo -e "${GREEN}[+]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[x]${NC} $1"; exit 1; }

# Check required tools
command -v aws >/dev/null 2>&1 || error "aws CLI not found. Install: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html"

# Safety check: Only allow localhost:4566
if [[ "${AWS_ENDPOINT_URL:-}" != "http://localhost:4566" ]]; then
    error "AWS_ENDPOINT_URL must be http://localhost:4566 (got: ${AWS_ENDPOINT_URL:-<not set>})"
fi

# Set credentials for LocalStack
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-test}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"
export AWS_EC2_METADATA_DISABLED=true

# Common tags
DEMO_TAG="Key=Project,Value=claws-demo"
DEMO_TAG2="Key=Demo,Value=true"

aws_cmd() {
    aws --endpoint-url="${AWS_ENDPOINT_URL}" "$@"
}

# Wait for LocalStack to be ready
wait_localstack() {
    log "Waiting for LocalStack..."
    for _ in {1..30}; do
        if aws_cmd ec2 describe-vpcs --query 'Vpcs[0].VpcId' --output text 2>/dev/null; then
            log "LocalStack is ready"
            return 0
        fi
        sleep 1
    done
    error "LocalStack not ready after 30 seconds"
}

# ============================================
# VPC A: Production-like setup
# ============================================
create_vpc_a() {
    log "Creating VPC A (prod)..."
    VPC_A=$(aws_cmd ec2 create-vpc --cidr-block 10.0.0.0/16 --query 'Vpc.VpcId' --output text)
    aws_cmd ec2 create-tags --resources "$VPC_A" --tags Key=Name,Value=claws-demo-prod ${DEMO_TAG} ${DEMO_TAG2}
    
    # Internet Gateway
    IGW_A=$(aws_cmd ec2 create-internet-gateway --query 'InternetGateway.InternetGatewayId' --output text)
    aws_cmd ec2 create-tags --resources "$IGW_A" --tags Key=Name,Value=claws-demo-igw-prod ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 attach-internet-gateway --vpc-id "$VPC_A" --internet-gateway-id "$IGW_A"
    
    # Public Subnets
    SUBNET_A1=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.1.0/24 --availability-zone us-east-1a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A1" --tags Key=Name,Value=claws-demo-public-1a ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_A2=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.2.0/24 --availability-zone us-east-1b --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A2" --tags Key=Name,Value=claws-demo-public-1b ${DEMO_TAG} ${DEMO_TAG2}
    
    # Private Subnet
    SUBNET_A3=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.10.0/24 --availability-zone us-east-1a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A3" --tags Key=Name,Value=claws-demo-private-1a ${DEMO_TAG} ${DEMO_TAG2}
    
    # Route Table (public)
    RTB_A=$(aws_cmd ec2 create-route-table --vpc-id "$VPC_A" --query 'RouteTable.RouteTableId' --output text)
    aws_cmd ec2 create-tags --resources "$RTB_A" --tags Key=Name,Value=claws-demo-rt-public ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 create-route --route-table-id "$RTB_A" --destination-cidr-block 0.0.0.0/0 --gateway-id "$IGW_A" || true
    aws_cmd ec2 associate-route-table --route-table-id "$RTB_A" --subnet-id "$SUBNET_A1" || true
    aws_cmd ec2 associate-route-table --route-table-id "$RTB_A" --subnet-id "$SUBNET_A2" || true
    
    # Route Table (private)
    RTB_A_PRIV=$(aws_cmd ec2 create-route-table --vpc-id "$VPC_A" --query 'RouteTable.RouteTableId' --output text)
    aws_cmd ec2 create-tags --resources "$RTB_A_PRIV" --tags Key=Name,Value=claws-demo-rt-private ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 associate-route-table --route-table-id "$RTB_A_PRIV" --subnet-id "$SUBNET_A3" || true
    
    # Security Groups
    SG_WEB=$(aws_cmd ec2 create-security-group --group-name claws-demo-sg-web --description "Web tier" --vpc-id "$VPC_A" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_WEB" --tags Key=Name,Value=claws-demo-sg-web ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 authorize-security-group-ingress --group-id "$SG_WEB" --protocol tcp --port 80 --cidr 0.0.0.0/0 || true
    aws_cmd ec2 authorize-security-group-ingress --group-id "$SG_WEB" --protocol tcp --port 443 --cidr 0.0.0.0/0 || true
    
    SG_APP=$(aws_cmd ec2 create-security-group --group-name claws-demo-sg-app --description "App tier" --vpc-id "$VPC_A" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_APP" --tags Key=Name,Value=claws-demo-sg-app ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 authorize-security-group-ingress --group-id "$SG_APP" --protocol tcp --port 8080 --source-group "$SG_WEB" || true
    
    SG_DB=$(aws_cmd ec2 create-security-group --group-name claws-demo-sg-db --description "Database tier" --vpc-id "$VPC_A" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_DB" --tags Key=Name,Value=claws-demo-sg-db ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 authorize-security-group-ingress --group-id "$SG_DB" --protocol tcp --port 3306 --source-group "$SG_APP" || true
    
    # LocalStack accepts any AMI ID for EC2 simulation
    AMI="ami-12345678"
    
    log "Creating EC2 instances in VPC A..."
    
    # Web servers
    WEB1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_A1" --security-group-ids "$SG_WEB" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$WEB1" --tags Key=Name,Value=claws-demo-web-1 Key=Role,Value=web ${DEMO_TAG} ${DEMO_TAG2}
    
    WEB2=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_A2" --security-group-ids "$SG_WEB" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$WEB2" --tags Key=Name,Value=claws-demo-web-2 Key=Role,Value=web ${DEMO_TAG} ${DEMO_TAG2}
    
    # App server
    APP1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.small --subnet-id "$SUBNET_A1" --security-group-ids "$SG_APP" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$APP1" --tags Key=Name,Value=claws-demo-app-1 Key=Role,Value=app ${DEMO_TAG} ${DEMO_TAG2}
    
    # DB server
    DB1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.medium --subnet-id "$SUBNET_A3" --security-group-ids "$SG_DB" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$DB1" --tags Key=Name,Value=claws-demo-db-1 Key=Role,Value=db ${DEMO_TAG} ${DEMO_TAG2}
    
    log "VPC A created: $VPC_A"
}

# ============================================
# VPC B: Dev/Staging setup
# ============================================
create_vpc_b() {
    log "Creating VPC B (dev)..."
    VPC_B=$(aws_cmd ec2 create-vpc --cidr-block 10.1.0.0/16 --query 'Vpc.VpcId' --output text)
    aws_cmd ec2 create-tags --resources "$VPC_B" --tags Key=Name,Value=claws-demo-dev ${DEMO_TAG} ${DEMO_TAG2}
    
    # Internet Gateway
    IGW_B=$(aws_cmd ec2 create-internet-gateway --query 'InternetGateway.InternetGatewayId' --output text)
    aws_cmd ec2 create-tags --resources "$IGW_B" --tags Key=Name,Value=claws-demo-igw-dev ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 attach-internet-gateway --vpc-id "$VPC_B" --internet-gateway-id "$IGW_B"
    
    # Subnets
    SUBNET_B1=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_B" --cidr-block 10.1.1.0/24 --availability-zone us-east-1a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_B1" --tags Key=Name,Value=claws-demo-dev-1a ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_B2=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_B" --cidr-block 10.1.2.0/24 --availability-zone us-east-1b --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_B2" --tags Key=Name,Value=claws-demo-dev-1b ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_B3=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_B" --cidr-block 10.1.3.0/24 --availability-zone us-east-1c --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_B3" --tags Key=Name,Value=claws-demo-dev-1c ${DEMO_TAG} ${DEMO_TAG2}
    
    # Route Table
    RTB_B=$(aws_cmd ec2 create-route-table --vpc-id "$VPC_B" --query 'RouteTable.RouteTableId' --output text)
    aws_cmd ec2 create-tags --resources "$RTB_B" --tags Key=Name,Value=claws-demo-rt-dev ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 create-route --route-table-id "$RTB_B" --destination-cidr-block 0.0.0.0/0 --gateway-id "$IGW_B" || true
    
    # Security Groups
    SG_DEV=$(aws_cmd ec2 create-security-group --group-name claws-demo-sg-dev --description "Dev all-in-one" --vpc-id "$VPC_B" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_DEV" --tags Key=Name,Value=claws-demo-sg-dev ${DEMO_TAG} ${DEMO_TAG2}
    
    SG_STAGING=$(aws_cmd ec2 create-security-group --group-name claws-demo-sg-staging --description "Staging" --vpc-id "$VPC_B" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_STAGING" --tags Key=Name,Value=claws-demo-sg-staging ${DEMO_TAG} ${DEMO_TAG2}
    
    SG_BASTION=$(aws_cmd ec2 create-security-group --group-name claws-demo-sg-bastion --description "Bastion host" --vpc-id "$VPC_B" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_BASTION" --tags Key=Name,Value=claws-demo-sg-bastion ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 authorize-security-group-ingress --group-id "$SG_BASTION" --protocol tcp --port 22 --cidr 0.0.0.0/0 || true
    
    # EC2 Instances
    AMI="ami-12345678"
    
    log "Creating EC2 instances in VPC B..."
    
    DEV1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_B1" --security-group-ids "$SG_DEV" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$DEV1" --tags Key=Name,Value=claws-demo-dev-server Key=Role,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    STAGING1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.small --subnet-id "$SUBNET_B2" --security-group-ids "$SG_STAGING" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$STAGING1" --tags Key=Name,Value=claws-demo-staging Key=Role,Value=staging ${DEMO_TAG} ${DEMO_TAG2}
    
    log "VPC B created: $VPC_B"
}

# ============================================
# S3 Buckets (for variety in demo)
# ============================================
create_s3_buckets() {
    log "Creating S3 buckets..."
    aws_cmd s3 mb s3://claws-demo-assets 2>/dev/null || true
    aws_cmd s3 mb s3://claws-demo-logs 2>/dev/null || true
    aws_cmd s3 mb s3://claws-demo-backups 2>/dev/null || true
}

# ============================================
# Main
# ============================================
main() {
    log "=== claws LocalStack Demo Setup ==="
    log "Endpoint: ${AWS_ENDPOINT_URL}"
    log "Region: ${AWS_DEFAULT_REGION}"
    
    wait_localstack
    
    # Clean up any existing demo resources first
    if [ -x "$(dirname "$0")/localstack-demo-cleanup.sh" ]; then
        warn "Running cleanup first..."
        "$(dirname "$0")/localstack-demo-cleanup.sh" 2>/dev/null || true
    fi
    
    create_vpc_a
    create_vpc_b
    create_s3_buckets
    
    log "=== Demo setup complete! ==="
    log ""
    log "Resources created:"
    log "  - 2 VPCs (prod, dev)"
    log "  - 6 Subnets"
    log "  - 4 Route Tables"
    log "  - 2 Internet Gateways"
    log "  - 6 Security Groups"
    log "  - 6 EC2 Instances"
    log "  - 3 S3 Buckets"
    log ""
    log "Run claws with:"
    log "  AWS_ENDPOINT_URL=http://localhost:4566 ./claws"
}

main "$@"
