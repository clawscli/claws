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

# Check if stack already exists
if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" &>/dev/null; then
    echo "⚠️  Stack '$STACK_NAME' already exists. Updating..."

    aws cloudformation update-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://eks-test-stack.yaml \
        --parameters \
            ParameterKey=ClusterName,ParameterValue="$CLUSTER_NAME" \
        --capabilities CAPABILITY_IAM \
        --region "$REGION" || {
            if [[ $? -eq 254 ]]; then
                echo "ℹ️  No updates to be performed"
                exit 0
            else
                exit 1
            fi
        }

    echo "Waiting for stack update to complete..."
    aws cloudformation wait stack-update-complete \
        --stack-name "$STACK_NAME" \
        --region "$REGION"
else
    echo "Creating new stack..."

    aws cloudformation create-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://eks-test-stack.yaml \
        --parameters \
            ParameterKey=ClusterName,ParameterValue="$CLUSTER_NAME" \
        --capabilities CAPABILITY_IAM \
        --region "$REGION"

    echo "Waiting for stack creation to complete (this takes ~15-20 minutes)..."
    aws cloudformation wait stack-create-complete \
        --stack-name "$STACK_NAME" \
        --region "$REGION"
fi

echo ""
echo "✅ Stack deployment complete!"
echo ""
echo "Cluster Details:"
aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`ClusterName` || OutputKey==`ClusterEndpoint`].[OutputKey,OutputValue]' \
    --output table

echo ""
echo "To configure kubectl:"
echo "  aws eks update-kubeconfig --name $CLUSTER_NAME --region $REGION"
echo ""
echo "To test with claws:"
echo "  AWS_REGION=$REGION claws"
echo ""
