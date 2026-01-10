#!/bin/bash
set -e

STACK_NAME="${STACK_NAME:-claws-eks-test}"
REGION="${AWS_REGION:-us-east-1}"

echo "=== Cleaning up EKS Test Stack ==="
echo "Stack Name: $STACK_NAME"
echo "Region: $REGION"
echo ""

# Check if stack exists
if ! aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" &>/dev/null; then
    echo "ℹ️  Stack '$STACK_NAME' does not exist. Nothing to clean up."
    exit 0
fi

# Confirm deletion
read -p "⚠️  This will delete the entire EKS cluster and all resources. Continue? (yes/no): " confirm
if [[ "$confirm" != "yes" ]]; then
    echo "Aborted."
    exit 0
fi

echo "Deleting stack..."
aws cloudformation delete-stack \
    --stack-name "$STACK_NAME" \
    --region "$REGION"

echo "Waiting for stack deletion to complete (this takes ~10-15 minutes)..."
aws cloudformation wait stack-delete-complete \
    --stack-name "$STACK_NAME" \
    --region "$REGION"

echo ""
echo "✅ Stack deletion complete!"
echo ""
