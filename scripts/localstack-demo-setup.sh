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
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"
export AWS_EC2_METADATA_DISABLED=true

# Account IDs (LocalStack uses AWS_ACCESS_KEY_ID as account ID if 12 digits)
ACCOUNT_PROD="111111111111"
ACCOUNT_DEV="222222222222"

# Regions to create resources in
REGIONS=("us-east-1" "us-west-2" "ap-northeast-1")

# Common tags
DEMO_TAG="Key=Project,Value=claws-demo"
DEMO_TAG2="Key=Demo,Value=true"

aws_cmd() {
    local account="${AWS_ACCOUNT_ID:-$ACCOUNT_PROD}"
    local region="${AWS_REGION:-us-east-1}"
    AWS_ACCESS_KEY_ID="$account" aws --endpoint-url="${AWS_ENDPOINT_URL}" --region "$region" "$@"
}

aws_cmd_account() {
    local account="$1"
    shift
    local region="${AWS_REGION:-us-east-1}"
    AWS_ACCESS_KEY_ID="$account" aws --endpoint-url="${AWS_ENDPOINT_URL}" --region "$region" "$@"
}

aws_cmd_region() {
    local region="$1"
    shift
    local account="${AWS_ACCOUNT_ID:-$ACCOUNT_PROD}"
    AWS_ACCESS_KEY_ID="$account" aws --endpoint-url="${AWS_ENDPOINT_URL}" --region "$region" "$@"
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
# VPC A: Production-like setup (Account: prod, Region: us-east-1)
# ============================================
create_vpc_a() {
    log "Creating VPC A (prod) in us-east-1..."
    export AWS_ACCOUNT_ID="$ACCOUNT_PROD"
    export AWS_REGION="us-east-1"
    
    VPC_A=$(aws_cmd ec2 create-vpc --cidr-block 10.0.0.0/16 --query 'Vpc.VpcId' --output text)
    aws_cmd ec2 create-tags --resources "$VPC_A" --tags Key=Name,Value=prod-vpc-east Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    IGW_A=$(aws_cmd ec2 create-internet-gateway --query 'InternetGateway.InternetGatewayId' --output text)
    aws_cmd ec2 create-tags --resources "$IGW_A" --tags Key=Name,Value=prod-igw-east Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 attach-internet-gateway --vpc-id "$VPC_A" --internet-gateway-id "$IGW_A"
    
    SUBNET_A1=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.1.0/24 --availability-zone us-east-1a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A1" --tags Key=Name,Value=prod-public-web-1a Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_A2=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.2.0/24 --availability-zone us-east-1b --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A2" --tags Key=Name,Value=prod-public-web-1b Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_A3=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.10.0/24 --availability-zone us-east-1a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A3" --tags Key=Name,Value=prod-private-db-1a Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_A4=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_A" --cidr-block 10.0.11.0/24 --availability-zone us-east-1b --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_A4" --tags Key=Name,Value=prod-private-db-1b Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
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
    
    # Web servers (running)
    WEB1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_A1" --security-group-ids "$SG_WEB" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$WEB1" --tags Key=Name,Value=prod-web-1 Key=Role,Value=web Key=Status,Value=active ${DEMO_TAG} ${DEMO_TAG2}
    
    WEB2=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_A2" --security-group-ids "$SG_WEB" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$WEB2" --tags Key=Name,Value=prod-web-2 Key=Role,Value=web Key=Status,Value=active ${DEMO_TAG} ${DEMO_TAG2}
    
    # App server (stopped - maintenance)
    APP1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.small --subnet-id "$SUBNET_A1" --security-group-ids "$SG_APP" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$APP1" --tags Key=Name,Value=prod-app-1 Key=Role,Value=app Key=Status,Value=maintenance ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 stop-instances --instance-ids "$APP1" || true
    
    # DB server (running)
    DB1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.medium --subnet-id "$SUBNET_A3" --security-group-ids "$SG_DB" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$DB1" --tags Key=Name,Value=prod-db-1 Key=Role,Value=db Key=Status,Value=active ${DEMO_TAG} ${DEMO_TAG2}
    
    log "VPC A created: $VPC_A"
}

# ============================================
# VPC B: Dev Account (Account: dev, Region: us-west-2)
# ============================================
create_vpc_b() {
    log "Creating VPC B (dev) in us-west-2..."
    export AWS_ACCOUNT_ID="$ACCOUNT_DEV"
    export AWS_REGION="us-west-2"
    
    VPC_B=$(aws_cmd ec2 create-vpc --cidr-block 10.1.0.0/16 --query 'Vpc.VpcId' --output text)
    aws_cmd ec2 create-tags --resources "$VPC_B" --tags Key=Name,Value=dev-vpc-west Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    IGW_B=$(aws_cmd ec2 create-internet-gateway --query 'InternetGateway.InternetGatewayId' --output text)
    aws_cmd ec2 create-tags --resources "$IGW_B" --tags Key=Name,Value=dev-igw-west Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 attach-internet-gateway --vpc-id "$VPC_B" --internet-gateway-id "$IGW_B"
    
    SUBNET_B1=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_B" --cidr-block 10.1.1.0/24 --availability-zone us-west-2a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_B1" --tags Key=Name,Value=dev-web-west-2a Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    SUBNET_B2=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_B" --cidr-block 10.1.2.0/24 --availability-zone us-west-2b --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_B2" --tags Key=Name,Value=dev-db-west-2b Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    RTB_B=$(aws_cmd ec2 create-route-table --vpc-id "$VPC_B" --query 'RouteTable.RouteTableId' --output text)
    aws_cmd ec2 create-tags --resources "$RTB_B" --tags Key=Name,Value=dev-rt-west Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 create-route --route-table-id "$RTB_B" --destination-cidr-block 0.0.0.0/0 --gateway-id "$IGW_B" || true
    
    SG_DEV=$(aws_cmd ec2 create-security-group --group-name dev-sg-all --description "Dev all-in-one" --vpc-id "$VPC_B" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_DEV" --tags Key=Name,Value=dev-sg-all Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    AMI="ami-12345678"
    
    log "Creating EC2 instances in dev account (us-west-2)..."
    
    DEV1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_B1" --security-group-ids "$SG_DEV" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$DEV1" --tags Key=Name,Value=dev-api-server Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    DEV2=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.small --subnet-id "$SUBNET_B2" --security-group-ids "$SG_DEV" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$DEV2" --tags Key=Name,Value=dev-worker Key=Env,Value=dev ${DEMO_TAG} ${DEMO_TAG2}
    
    log "VPC B created: $VPC_B"
}

# ============================================
# VPC C: Prod Account in ap-northeast-1
# ============================================
create_vpc_c() {
    log "Creating VPC C (prod) in ap-northeast-1..."
    export AWS_ACCOUNT_ID="$ACCOUNT_PROD"
    export AWS_REGION="ap-northeast-1"
    
    VPC_C=$(aws_cmd ec2 create-vpc --cidr-block 10.2.0.0/16 --query 'Vpc.VpcId' --output text)
    aws_cmd ec2 create-tags --resources "$VPC_C" --tags Key=Name,Value=prod-vpc-tokyo Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    IGW_C=$(aws_cmd ec2 create-internet-gateway --query 'InternetGateway.InternetGatewayId' --output text)
    aws_cmd ec2 create-tags --resources "$IGW_C" --tags Key=Name,Value=prod-igw-tokyo Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    aws_cmd ec2 attach-internet-gateway --vpc-id "$VPC_C" --internet-gateway-id "$IGW_C"
    
    SUBNET_C1=$(aws_cmd ec2 create-subnet --vpc-id "$VPC_C" --cidr-block 10.2.1.0/24 --availability-zone ap-northeast-1a --query 'Subnet.SubnetId' --output text)
    aws_cmd ec2 create-tags --resources "$SUBNET_C1" --tags Key=Name,Value=prod-web-tokyo-1a Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    SG_TOKYO=$(aws_cmd ec2 create-security-group --group-name prod-sg-tokyo --description "Tokyo prod" --vpc-id "$VPC_C" --query 'GroupId' --output text)
    aws_cmd ec2 create-tags --resources "$SG_TOKYO" --tags Key=Name,Value=prod-sg-tokyo Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    AMI="ami-12345678"
    
    log "Creating EC2 instances in prod account (ap-northeast-1)..."
    
    TOKYO1=$(aws_cmd ec2 run-instances --image-id "$AMI" --instance-type t2.micro --subnet-id "$SUBNET_C1" --security-group-ids "$SG_TOKYO" --query 'Instances[0].InstanceId' --output text)
    aws_cmd ec2 create-tags --resources "$TOKYO1" --tags Key=Name,Value=prod-tokyo-web Key=Env,Value=prod ${DEMO_TAG} ${DEMO_TAG2}
    
    log "VPC C created: $VPC_C"
}

# ============================================
# S3 Buckets
# ============================================
create_s3_buckets() {
    log "Creating S3 buckets..."
    export AWS_ACCOUNT_ID="$ACCOUNT_PROD"
    export AWS_REGION="us-east-1"
    aws_cmd s3 mb s3://claws-demo-assets 2>/dev/null || true
    aws_cmd s3 mb s3://claws-demo-logs 2>/dev/null || true
    aws_cmd s3 mb s3://claws-demo-backups 2>/dev/null || true
}

# ============================================
# AWS Config for multi-profile demo
# ============================================
create_aws_config() {
    log "Creating AWS config for demo profiles..."
    
    CONFIG_DIR="$(dirname "$0")/demo-aws-config"
    mkdir -p "$CONFIG_DIR"
    
    cat > "$CONFIG_DIR/config" << 'AWSCONFIG'
[default]
region = us-east-1
endpoint_url = http://localhost:4566

[profile prod]
region = us-east-1
endpoint_url = http://localhost:4566

[profile dev]
region = us-west-2
endpoint_url = http://localhost:4566
AWSCONFIG

    cat > "$CONFIG_DIR/credentials" << 'AWSCREDS'
[default]
aws_access_key_id = 111111111111
aws_secret_access_key = test

[prod]
aws_access_key_id = 111111111111
aws_secret_access_key = test

[dev]
aws_access_key_id = 222222222222
aws_secret_access_key = test
AWSCREDS

    chmod 600 "$CONFIG_DIR/credentials"
    log "AWS config created at: $CONFIG_DIR"
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
    create_vpc_c
    create_s3_buckets
    create_aws_config
    
    log "=== Demo setup complete! ==="
    log ""
    log "Resources created:"
    log "  - 3 VPCs (prod us-east-1, dev us-west-2, prod ap-northeast-1)"
    log "  - 2 Accounts (111111111111=prod, 222222222222=dev)"
    log "  - 3 Regions (us-east-1, us-west-2, ap-northeast-1)"
    log "  - 7 EC2 Instances across accounts/regions"
    log "  - 3 S3 Buckets"
    log ""
    log "AWS config created at: scripts/demo-aws-config/"
    log ""
    log "Run claws with:"
    log "  AWS_ENDPOINT_URL=http://localhost:4566 ./claws"
}

main "$@"
