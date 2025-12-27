#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

YELLOW='\033[0;33m'

log() { echo -e "${GREEN}[+]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[x]${NC} $1"; exit 1; }

command -v aws >/dev/null 2>&1 || error "aws CLI not found"

ERRORS=0
track_error() { ((ERRORS++)) || true; warn "$1"; }

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
    --filters "Name=tag:Project,Values=claws-demo" "Name=instance-state-name,Values=pending,running,stopping,stopped" \
    --query 'Reservations[].Instances[].InstanceId' --output text 2>/dev/null || echo "")
if [[ -n "$INSTANCES" ]]; then
    for id in $INSTANCES; do
        if aws_cmd ec2 terminate-instances --instance-ids "$id" 2>/dev/null; then
            log "  Terminating: $id"
        else
            track_error "Failed to terminate: $id"
        fi
    done
    log "  Waiting for instances to terminate..."
    for i in $(seq 1 30); do
        # shellcheck disable=SC2086 # Word splitting intended for multiple IDs
        REMAINING=$(aws_cmd ec2 describe-instances \
            --instance-ids $INSTANCES \
            --filters "Name=instance-state-name,Values=pending,running,shutting-down,stopping,stopped" \
            --query 'Reservations[].Instances[].InstanceId' --output text 2>/dev/null || echo "")
        if [[ -z "$REMAINING" ]]; then
            log "  All instances terminated"
            break
        fi
        if [[ $i -eq 30 ]]; then
            track_error "Timeout waiting for instances to terminate"
        fi
        sleep 1
    done
fi

log "Deleting Security Groups..."
SGS=$(aws_cmd ec2 describe-security-groups \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'SecurityGroups[].GroupId' --output text 2>/dev/null || echo "")
for sg in $SGS; do
    if aws_cmd ec2 delete-security-group --group-id "$sg" 2>/dev/null; then
        log "  Deleted: $sg"
    else
        track_error "Failed to delete SG: $sg"
    fi
done

log "Deleting Subnets..."
SUBNETS=$(aws_cmd ec2 describe-subnets \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'Subnets[].SubnetId' --output text 2>/dev/null || echo "")
for subnet in $SUBNETS; do
    if aws_cmd ec2 delete-subnet --subnet-id "$subnet" 2>/dev/null; then
        log "  Deleted: $subnet"
    else
        track_error "Failed to delete subnet: $subnet"
    fi
done

log "Deleting Route Tables..."
RTS=$(aws_cmd ec2 describe-route-tables \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'RouteTables[].RouteTableId' --output text 2>/dev/null || echo "")
for rt in $RTS; do
    ASSOCS=$(aws_cmd ec2 describe-route-tables --route-table-ids "$rt" \
        --query 'RouteTables[0].Associations[?!Main].RouteTableAssociationId' --output text 2>/dev/null || echo "")
    for assoc in $ASSOCS; do
        aws_cmd ec2 disassociate-route-table --association-id "$assoc" 2>/dev/null || track_error "Failed to disassociate: $assoc"
    done
    if aws_cmd ec2 delete-route-table --route-table-id "$rt" 2>/dev/null; then
        log "  Deleted: $rt"
    else
        track_error "Failed to delete RT: $rt"
    fi
done

log "Detaching and deleting Internet Gateways..."
IGWS=$(aws_cmd ec2 describe-internet-gateways \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'InternetGateways[].InternetGatewayId' --output text 2>/dev/null || echo "")
for igw in $IGWS; do
    VPC=$(aws_cmd ec2 describe-internet-gateways --internet-gateway-ids "$igw" \
        --query 'InternetGateways[0].Attachments[0].VpcId' --output text 2>/dev/null || echo "")
    if [[ -n "$VPC" && "$VPC" != "None" ]]; then
        aws_cmd ec2 detach-internet-gateway --internet-gateway-id "$igw" --vpc-id "$VPC" 2>/dev/null || track_error "Failed to detach IGW: $igw"
    fi
    if aws_cmd ec2 delete-internet-gateway --internet-gateway-id "$igw" 2>/dev/null; then
        log "  Deleted: $igw"
    else
        track_error "Failed to delete IGW: $igw"
    fi
done

log "Deleting VPCs..."
VPCS=$(aws_cmd ec2 describe-vpcs \
    --filters "Name=tag:Project,Values=claws-demo" \
    --query 'Vpcs[].VpcId' --output text 2>/dev/null || echo "")
for vpc in $VPCS; do
    if aws_cmd ec2 delete-vpc --vpc-id "$vpc" 2>/dev/null; then
        log "  Deleted: $vpc"
    else
        track_error "Failed to delete VPC: $vpc"
    fi
done

log "Deleting S3 buckets..."
for bucket in claws-demo-assets claws-demo-logs claws-demo-backups; do
    if aws_cmd s3 rb "s3://${bucket}" --force 2>/dev/null; then
        log "  Deleted: $bucket"
    else
        track_error "Failed to delete bucket: $bucket"
    fi
done

if [[ $ERRORS -gt 0 ]]; then
    warn "=== Cleanup completed with $ERRORS error(s) ==="
    exit 1
else
    log "=== Cleanup complete ==="
fi
