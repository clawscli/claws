#!/bin/bash
set -e

STACK_NAME="${STACK_NAME:-claws-eks-test}"
CLUSTER_NAME="${CLUSTER_NAME:-claws-test-cluster}"
REGION="${AWS_REGION:-us-east-1}"

echo "=== Deploying EKS Test Stack ==="
echo "Stack Name: $STACK_NAME"
echo "Cluster Name: $CLUSTER_NAME"
echo "Region: $REGION"
echo ""

# Check if stack exists
STACK_EXISTS=false
if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" &>/dev/null; then
    STACK_EXISTS=true
    STACK_STATUS=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" --query 'Stacks[0].StackStatus' --output text)
    echo "Stack Status: $STACK_STATUS"
fi

# Handle existing stack in failed state
if [ "$STACK_EXISTS" = true ]; then
    case "$STACK_STATUS" in
        *ROLLBACK_COMPLETE|*FAILED)
            echo "⚠️  Stack in failed state. Deleting before recreate..."
            aws cloudformation delete-stack --stack-name "$STACK_NAME" --region "$REGION"
            echo "Waiting for deletion..."
            aws cloudformation wait stack-delete-complete --stack-name "$STACK_NAME" --region "$REGION" || true
            STACK_EXISTS=false
            ;;
        *IN_PROGRESS)
            echo "❌ Stack operation in progress. Wait or cancel existing operation."
            exit 1
            ;;
    esac
fi

# Deploy stack
if [ "$STACK_EXISTS" = true ]; then
    echo "⚠️  Stack exists. Updating..."

    UPDATE_OUTPUT=$(aws cloudformation update-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://eks-test-stack.yaml \
        --parameters \
            ParameterKey=ClusterName,ParameterValue="$CLUSTER_NAME" \
        --capabilities CAPABILITY_IAM \
        --region "$REGION" 2>&1) || UPDATE_EXIT=$?

    if [ ${UPDATE_EXIT:-0} -ne 0 ]; then
        if echo "$UPDATE_OUTPUT" | grep -q "No updates are to be performed"; then
            echo "ℹ️  No updates needed"
        else
            echo "❌ Update failed: $UPDATE_OUTPUT"
            exit 1
        fi
    else
        echo "Waiting for update (~15-20 min)..."
        aws cloudformation wait stack-update-complete \
            --stack-name "$STACK_NAME" \
            --region "$REGION"
    fi
else
    echo "Creating stack..."

    aws cloudformation create-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://eks-test-stack.yaml \
        --parameters \
            ParameterKey=ClusterName,ParameterValue="$CLUSTER_NAME" \
        --capabilities CAPABILITY_IAM \
        --region "$REGION"

    echo "Waiting for creation (~15-20 min)..."
    aws cloudformation wait stack-create-complete \
        --stack-name "$STACK_NAME" \
        --region "$REGION"
fi

# Verify deployment
echo ""
echo "✅ Stack deployed!"
echo ""

# Verify cluster is active
CLUSTER_STATUS=$(aws eks describe-cluster --name "$CLUSTER_NAME" --region "$REGION" --query 'cluster.status' --output text 2>/dev/null || echo "NOT_FOUND")
if [ "$CLUSTER_STATUS" = "ACTIVE" ]; then
    echo "✓ EKS Cluster: ACTIVE"
else
    echo "⚠️  EKS Cluster: $CLUSTER_STATUS"
fi

# Verify auth mode
AUTH_MODE=$(aws eks describe-cluster --name "$CLUSTER_NAME" --region "$REGION" --query 'cluster.accessConfig.authenticationMode' --output text 2>/dev/null || echo "UNKNOWN")
echo "✓ Auth Mode: $AUTH_MODE"

# Show outputs
echo ""
echo "Stack Outputs:"
aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`ClusterName` || OutputKey==`ClusterEndpoint`].[OutputKey,OutputValue]' \
    --output table

echo ""
echo "Next steps:"
echo "  # Configure kubectl:"
echo "  aws eks update-kubeconfig --name $CLUSTER_NAME --region $REGION"
echo ""
echo "  # Test with claws:"
echo "  AWS_REGION=$REGION claws"
echo ""
