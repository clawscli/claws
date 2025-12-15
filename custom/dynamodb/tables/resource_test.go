package tables

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestNewTableResource(t *testing.T) {
	creationTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	table := types.TableDescription{
		TableName:        aws.String("test-table"),
		TableArn:         aws.String("arn:aws:dynamodb:us-east-1:123456789012:table/test-table"),
		TableId:          aws.String("abc123-def456"),
		TableStatus:      types.TableStatusActive,
		ItemCount:        aws.Int64(1000),
		TableSizeBytes:   aws.Int64(1048576), // 1MB
		CreationDateTime: &creationTime,
		BillingModeSummary: &types.BillingModeSummary{
			BillingMode: types.BillingModePayPerRequest,
		},
		ProvisionedThroughput: &types.ProvisionedThroughputDescription{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		DeletionProtectionEnabled: aws.Bool(true),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndexDescription{
			{IndexName: aws.String("gsi1")},
			{IndexName: aws.String("gsi2")},
		},
		LocalSecondaryIndexes: []types.LocalSecondaryIndexDescription{
			{IndexName: aws.String("lsi1")},
		},
		LatestStreamArn: aws.String("arn:aws:dynamodb:us-east-1:123456789012:table/test-table/stream/2024-01-15"),
	}

	resource := NewTableResource(table)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "test-table"},
		{"GetName", resource.GetName(), "test-table"},
		{"GetARN", resource.GetARN(), "arn:aws:dynamodb:us-east-1:123456789012:table/test-table"},
		{"Status", resource.Status(), "ACTIVE"},
		{"ItemCount", resource.ItemCount(), int64(1000)},
		{"SizeBytes", resource.SizeBytes(), int64(1048576)},
		{"BillingMode", resource.BillingMode(), "PAY_PER_REQUEST"},
		{"ReadCapacity", resource.ReadCapacity(), int64(5)},
		{"WriteCapacity", resource.WriteCapacity(), int64(5)},
		{"GSICount", resource.GSICount(), 2},
		{"LSICount", resource.LSICount(), 1},
		{"DeletionProtectionEnabled", resource.DeletionProtectionEnabled(), true},
		{"TableId", resource.TableId(), "abc123-def456"},
		{"StreamArn", resource.StreamArn(), "arn:aws:dynamodb:us-east-1:123456789012:table/test-table/stream/2024-01-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test KeySchema
	keySchema := resource.KeySchema()
	if len(keySchema) != 2 {
		t.Errorf("KeySchema length = %d, want 2", len(keySchema))
	}

	// Test CreationDateTime format
	createdAt := resource.CreationDateTime()
	if createdAt != "2024-01-15 10:30:00" {
		t.Errorf("CreationDateTime() = %q, want %q", createdAt, "2024-01-15 10:30:00")
	}
}

func TestTableResource_MinimalTable(t *testing.T) {
	table := types.TableDescription{
		TableName:   aws.String("minimal-table"),
		TableStatus: types.TableStatusCreating,
	}

	resource := NewTableResource(table)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "minimal-table"},
		{"Status", resource.Status(), "CREATING"},
		{"ItemCount", resource.ItemCount(), int64(0)},
		{"SizeBytes", resource.SizeBytes(), int64(0)},
		{"BillingMode", resource.BillingMode(), "PROVISIONED"}, // Default
		{"ReadCapacity", resource.ReadCapacity(), int64(0)},
		{"WriteCapacity", resource.WriteCapacity(), int64(0)},
		{"GSICount", resource.GSICount(), 0},
		{"LSICount", resource.LSICount(), 0},
		{"DeletionProtectionEnabled", resource.DeletionProtectionEnabled(), false},
		{"TableId", resource.TableId(), ""},
		{"StreamArn", resource.StreamArn(), ""},
		{"CreationDateTime", resource.CreationDateTime(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestTableResource_TableClass(t *testing.T) {
	tests := []struct {
		name     string
		class    *types.TableClassSummary
		expected string
	}{
		{"nil class", nil, "STANDARD"},
		{"STANDARD", &types.TableClassSummary{TableClass: types.TableClassStandard}, "STANDARD"},
		{"STANDARD_IA", &types.TableClassSummary{TableClass: types.TableClassStandardInfrequentAccess}, "STANDARD_INFREQUENT_ACCESS"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			table := types.TableDescription{
				TableName:         aws.String("test"),
				TableClassSummary: tc.class,
			}
			resource := NewTableResource(table)
			if got := resource.TableClass(); got != tc.expected {
				t.Errorf("TableClass() = %q, want %q", got, tc.expected)
			}
		})
	}
}
