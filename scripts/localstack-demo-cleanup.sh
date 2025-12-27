#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log() { echo -e "${GREEN}[+]${NC} $1"; }
error() { echo -e "${RED}[x]${NC} $1"; exit 1; }

if [[ "${AWS_ENDPOINT_URL:-}" != "http://localhost:4566" ]]; then
    error "AWS_ENDPOINT_URL must be http://localhost:4566 (got: ${AWS_ENDPOINT_URL:-<not set>})"
fi

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-test}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"
export AWS_EC2_METADATA_DISABLED=true

aws_cmd() {
    aws --endpoint-url="${AWS_ENDPOINT_URL}" "$@"
}

log "=== claws LocalStack Demo Cleanup ==="

log "Terminating EC2 instances..."
INSTANCES=$(aws_cmd ec2 describe-instances \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'Reservations[].Instances[].InstanceId' --output text 2>/dev/null || echo "")
if [[ -n "$INSTANCES" ]]; then
    for id in $INSTANCES; do
        aws_cmd ec2 terminate-instances --instance-ids "$id" 2>/dev/null || true
        log "  Terminated: $id"
    done
    sleep 2
fi

log "Deleting Security Groups..."
SGS=$(aws_cmd ec2 describe-security-groups \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'SecurityGroups[].GroupId' --output text 2>/dev/null || echo "")
for sg in $SGS; do
    aws_cmd ec2 delete-security-group --group-id "$sg" 2>/dev/null || true
    log "  Deleted: $sg"
done

log "Deleting Subnets..."
SUBNETS=$(aws_cmd ec2 describe-subnets \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'Subnets[].SubnetId' --output text 2>/dev/null || echo "")
for subnet in $SUBNETS; do
    aws_cmd ec2 delete-subnet --subnet-id "$subnet" 2>/dev/null || true
    log "  Deleted: $subnet"
done

log "Deleting Route Tables..."
RTS=$(aws_cmd ec2 describe-route-tables \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'RouteTables[].RouteTableId' --output text 2>/dev/null || echo "")
for rt in $RTS; do
    ASSOCS=$(aws_cmd ec2 describe-route-tables --route-table-ids "$rt" \
        --query 'RouteTables[0].Associations[?!Main].RouteTableAssociationId' --output text 2>/dev/null || echo "")
    for assoc in $ASSOCS; do
        aws_cmd ec2 disassociate-route-table --association-id "$assoc" 2>/dev/null || true
    done
    aws_cmd ec2 delete-route-table --route-table-id "$rt" 2>/dev/null || true
    log "  Deleted: $rt"
done

log "Detaching and deleting Internet Gateways..."
IGWS=$(aws_cmd ec2 describe-internet-gateways \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'InternetGateways[].InternetGatewayId' --output text 2>/dev/null || echo "")
for igw in $IGWS; do
    VPC=$(aws_cmd ec2 describe-internet-gateways --internet-gateway-ids "$igw" \
        --query 'InternetGateways[0].Attachments[0].VpcId' --output text 2>/dev/null || echo "")
    if [[ -n "$VPC" && "$VPC" != "None" ]]; then
        aws_cmd ec2 detach-internet-gateway --internet-gateway-id "$igw" --vpc-id "$VPC" 2>/dev/null || true
    fi
    aws_cmd ec2 delete-internet-gateway --internet-gateway-id "$igw" 2>/dev/null || true
    log "  Deleted: $igw"
done

log "Deleting VPCs..."
VPCS=$(aws_cmd ec2 describe-vpcs \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'Vpcs[].VpcId' --output text 2>/dev/null || echo "")
for vpc in $VPCS; do
    aws_cmd ec2 delete-vpc --vpc-id "$vpc" 2>/dev/null || true
    log "  Deleted: $vpc"
done

log "Deleting S3 buckets..."
for bucket in claws-demo-assets claws-demo-logs claws-demo-backups; do
    aws_cmd s3 rb "s3://${bucket}" --force 2>/dev/null || true
    log "  Deleted: $bucket"
done

log "=== Cleanup complete ==="
