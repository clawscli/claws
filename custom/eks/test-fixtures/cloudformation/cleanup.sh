#!/bin/bash
set -e

STACK_NAME="${STACK_NAME:-claws-eks-test}"
CLUSTER_NAME="${CLUSTER_NAME:-claws-test-cluster}"
REGION="${AWS_REGION:-us-east-1}"
CI_MODE="${CI_MODE:-false}"
MAX_RETRY=2

echo "=== Cleaning up EKS Test Stack ==="
echo "Stack Name: $STACK_NAME"
echo "Cluster Name: $CLUSTER_NAME"
echo "Region: $REGION"
echo ""

# Check if stack exists
if ! aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" &>/dev/null; then
    echo "ℹ️  Stack '$STACK_NAME' does not exist. Nothing to clean up."
    exit 0
fi

# Get current stack status
STACK_STATUS=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" --query 'Stacks[0].StackStatus' --output text)
echo "Stack Status: $STACK_STATUS"

# Confirm deletion (skip in CI mode)
if [ "$CI_MODE" != "true" ]; then
    read -p "⚠️  Delete entire EKS cluster and resources? (yes/no): " confirm
    if [[ "$confirm" != "yes" ]]; then
        echo "Aborted."
        exit 0
    fi
fi

# Function to check and fix cluster auth mode if needed
fix_auth_mode_if_needed() {
    local cluster_name=$1
    local region=$2

    # Check if cluster exists
    if ! aws eks describe-cluster --name "$cluster_name" --region "$region" &>/dev/null; then
        echo "ℹ️  Cluster already deleted or not found"
        return 0
    fi

    # Check auth mode
    AUTH_MODE=$(aws eks describe-cluster --name "$cluster_name" --region "$region" \
        --query 'cluster.accessConfig.authenticationMode' --output text 2>/dev/null || echo "CONFIG_MAP")

    echo "Current auth mode: $AUTH_MODE"

    if [ "$AUTH_MODE" = "CONFIG_MAP" ]; then
        echo "⚠️  Auth mode is CONFIG_MAP. Updating to API_AND_CONFIG_MAP..."

        UPDATE_ID=$(aws eks update-cluster-config \
            --name "$cluster_name" \
            --access-config authenticationMode=API_AND_CONFIG_MAP \
            --region "$region" \
            --query 'update.id' --output text)

        echo "Update ID: $UPDATE_ID"
        echo "Waiting for auth mode update (~2-3 min)..."

        # Wait for update to complete
        for i in {1..40}; do
            STATUS=$(aws eks describe-update \
                --name "$cluster_name" \
                --update-id "$UPDATE_ID" \
                --region "$region" \
                --query 'update.status' --output text 2>/dev/null || echo "Failed")

            echo "  [$i/40] Update status: $STATUS"

            if [ "$STATUS" = "Successful" ]; then
                echo "✓ Auth mode updated successfully"
                return 0
            elif [ "$STATUS" = "Failed" ]; then
                echo "❌ Auth mode update failed"
                return 1
            fi

            sleep 5
        done

        echo "⚠️  Auth mode update timeout"
        return 1
    fi

    return 0
}

# Delete stack with retry logic
delete_stack_with_retry() {
    local attempt=1

    while [ $attempt -le $MAX_RETRY ]; do
        echo ""
        echo "=== Deletion attempt $attempt/$MAX_RETRY ==="

        # Delete stack
        aws cloudformation delete-stack \
            --stack-name "$STACK_NAME" \
            --region "$REGION"

        echo "Waiting for deletion (~10-15 min)..."

        # Wait for deletion with timeout
        if aws cloudformation wait stack-delete-complete \
            --stack-name "$STACK_NAME" \
            --region "$REGION" 2>&1; then
            echo "✅ Stack deleted successfully"
            return 0
        fi

        # Check if stack still exists
        if ! aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" &>/dev/null; then
            echo "✅ Stack deleted successfully"
            return 0
        fi

        # Get current status
        CURRENT_STATUS=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" \
            --query 'Stacks[0].StackStatus' --output text)

        echo "Current status: $CURRENT_STATUS"

        if [ "$CURRENT_STATUS" = "DELETE_FAILED" ]; then
            echo "⚠️  Deletion failed. Checking for issues..."

            # Show failed resources
            echo "Failed resources:"
            aws cloudformation describe-stack-resources \
                --stack-name "$STACK_NAME" \
                --region "$REGION" \
                --query 'StackResources[?ResourceStatus==`DELETE_FAILED`].[LogicalResourceId,ResourceType,ResourceStatusReason]' \
                --output table

            # Check for EKSAccessEntry issues
            FAILED_RESOURCE=$(aws cloudformation describe-stack-resources \
                --stack-name "$STACK_NAME" \
                --region "$REGION" \
                --query 'StackResources[?ResourceStatus==`DELETE_FAILED` && ResourceType==`AWS::EKS::AccessEntry`].LogicalResourceId' \
                --output text)

            if [ -n "$FAILED_RESOURCE" ]; then
                echo "EKSAccessEntry deletion failed. Fixing auth mode..."

                if fix_auth_mode_if_needed "$CLUSTER_NAME" "$REGION"; then
                    echo "Retrying deletion..."
                    attempt=$((attempt + 1))
                    sleep 5
                    continue
                else
                    echo "❌ Cannot fix auth mode. Manual intervention required."
                    return 1
                fi
            fi

            # Other failures - retry anyway
            echo "Retrying deletion..."
            attempt=$((attempt + 1))
            sleep 5
        else
            echo "❌ Unexpected status: $CURRENT_STATUS"
            return 1
        fi
    done

    echo "❌ Max retries exceeded"
    return 1
}

# Execute deletion
if delete_stack_with_retry; then
    echo ""
    echo "✅ Cleanup complete!"
    echo ""
else
    echo ""
    echo "❌ Cleanup failed after $MAX_RETRY attempts"
    echo ""
    echo "Manual cleanup required:"
    echo "  1. Check CloudFormation console for detailed errors"
    echo "  2. Delete stuck resources manually"
    echo "  3. Run: aws cloudformation delete-stack --stack-name $STACK_NAME --region $REGION"
    echo ""
    exit 1
fi
