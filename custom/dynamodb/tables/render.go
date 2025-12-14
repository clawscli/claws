package tables

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TableRenderer renders DynamoDB tables
type TableRenderer struct {
	render.BaseRenderer
}

// NewTableRenderer creates a new TableRenderer
func NewTableRenderer() render.Renderer {
	return &TableRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "dynamodb",
			Resource: "tables",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "BILLING", Width: 12, Getter: getBillingMode},
				{Name: "ITEMS", Width: 12, Getter: getItemCount},
				{Name: "SIZE", Width: 10, Getter: getTableSize},
				{Name: "GSI", Width: 4, Getter: getGSICount},
				{Name: "RCU/WCU", Width: 10, Getter: getCapacity},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	if table, ok := r.(*TableResource); ok {
		status := table.Status()
		switch status {
		case "ACTIVE":
			return "active"
		case "CREATING":
			return "pending"
		case "UPDATING":
			return "pending"
		case "DELETING":
			return "deleting"
		case "ARCHIVED":
			return "stopped"
		default:
			return strings.ToLower(status)
		}
	}
	return ""
}

func getBillingMode(r dao.Resource) string {
	if table, ok := r.(*TableResource); ok {
		mode := table.BillingMode()
		switch mode {
		case "PAY_PER_REQUEST":
			return "on-demand"
		case "PROVISIONED":
			return "provisioned"
		default:
			return strings.ToLower(mode)
		}
	}
	return ""
}

func getItemCount(r dao.Resource) string {
	if table, ok := r.(*TableResource); ok {
		count := table.ItemCount()
		if count >= 1000000 {
			return fmt.Sprintf("%.1fM", float64(count)/1000000)
		}
		if count >= 1000 {
			return fmt.Sprintf("%.1fK", float64(count)/1000)
		}
		return fmt.Sprintf("%d", count)
	}
	return ""
}

func getTableSize(r dao.Resource) string {
	if table, ok := r.(*TableResource); ok {
		return render.FormatSize(table.SizeBytes())
	}
	return ""
}

func getGSICount(r dao.Resource) string {
	if table, ok := r.(*TableResource); ok {
		return fmt.Sprintf("%d", table.GSICount())
	}
	return ""
}

func getCapacity(r dao.Resource) string {
	if table, ok := r.(*TableResource); ok {
		if table.BillingMode() == "PAY_PER_REQUEST" {
			return "-"
		}
		return fmt.Sprintf("%d/%d", table.ReadCapacity(), table.WriteCapacity())
	}
	return ""
}

// RenderDetail renders detailed table information
func (r *TableRenderer) RenderDetail(resource dao.Resource) string {
	table, ok := resource.(*TableResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("DynamoDB Table", table.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", table.GetName())
	d.Field("ARN", table.GetARN())
	if tableId := table.TableId(); tableId != "" {
		d.Field("Table ID", tableId)
	}
	d.FieldStyled("Status", table.Status(), render.StateColorer()(strings.ToLower(table.Status())))
	d.Field("Table Class", table.TableClass())

	billing := table.BillingMode()
	if billing == "PAY_PER_REQUEST" {
		d.Field("Billing Mode", "On-Demand")
	} else {
		d.Field("Billing Mode", "Provisioned")
	}

	// Protection
	if table.DeletionProtectionEnabled() {
		d.Field("Deletion Protection", "Enabled")
	} else {
		d.Field("Deletion Protection", "Disabled")
	}

	// Statistics
	d.Section("Statistics")
	d.Field("Item Count", fmt.Sprintf("%d", table.ItemCount()))
	d.Field("Table Size", render.FormatSize(table.SizeBytes()))

	// Key Schema
	keySchema := table.KeySchema()
	if len(keySchema) > 0 {
		d.Section("Key Schema")
		for _, k := range keySchema {
			keyType := "HASH (Partition Key)"
			if k.KeyType == types.KeyTypeRange {
				keyType = "RANGE (Sort Key)"
			}
			if k.AttributeName != nil {
				d.Field(*k.AttributeName, keyType)
			}
		}
	}

	// Capacity
	if billing == "PROVISIONED" {
		d.Section("Provisioned Capacity")
		d.Field("Read Capacity", fmt.Sprintf("%d RCU", table.ReadCapacity()))
		d.Field("Write Capacity", fmt.Sprintf("%d WCU", table.WriteCapacity()))
	}

	// Global Secondary Indexes
	gsis := table.GlobalSecondaryIndexes()
	if len(gsis) > 0 {
		d.Section("Global Secondary Indexes")
		for _, gsi := range gsis {
			if gsi.IndexName == nil {
				continue
			}
			status := "UNKNOWN"
			if gsi.IndexStatus != "" {
				status = string(gsi.IndexStatus)
			}
			d.Field(*gsi.IndexName, status)
		}
	}

	// Server-Side Encryption
	if sse := table.SSEDescription(); sse != nil {
		d.Section("Server-Side Encryption")
		d.Field("Status", string(sse.Status))
		if sse.SSEType != "" {
			d.Field("Type", string(sse.SSEType))
		}
		if sse.KMSMasterKeyArn != nil {
			d.Field("KMS Key ARN", *sse.KMSMasterKeyArn)
		}
	}

	// Stream
	if stream := table.Item.StreamSpecification; stream != nil && stream.StreamEnabled != nil && *stream.StreamEnabled {
		d.Section("DynamoDB Streams")
		d.Field("Enabled", "Yes")
		if stream.StreamViewType != "" {
			d.Field("View Type", string(stream.StreamViewType))
		}
		if streamArn := table.StreamArn(); streamArn != "" {
			d.Field("Stream ARN", streamArn)
		}
	}

	// Global Table Replicas
	if replicas := table.Replicas(); len(replicas) > 0 {
		d.Section("Global Table Replicas")
		for _, replica := range replicas {
			d.Field(appaws.Str(replica.RegionName), string(replica.ReplicaStatus))
		}
	}

	// Restore Summary (if restored from backup)
	if restore := table.RestoreSummary(); restore != nil {
		d.Section("Restore Information")
		if restore.SourceBackupArn != nil {
			d.Field("Source Backup", *restore.SourceBackupArn)
		}
		if restore.SourceTableArn != nil {
			d.Field("Source Table", *restore.SourceTableArn)
		}
		if restore.RestoreDateTime != nil {
			d.Field("Restored At", restore.RestoreDateTime.Format("2006-01-02 15:04:05"))
		}
		d.Field("Restore In Progress", fmt.Sprintf("%v", restore.RestoreInProgress))
	}

	// Timestamps
	if created := table.CreationDateTime(); created != "" {
		d.Section("Timestamps")
		d.Field("Created", created)
	}

	// Tags
	d.Tags(table.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *TableRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	table, ok := resource.(*TableResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: table.GetName()},
		{Label: "ARN", Value: table.GetARN()},
		{Label: "Status", Value: table.Status()},
		{Label: "Billing Mode", Value: table.BillingMode()},
		{Label: "Items", Value: fmt.Sprintf("%d", table.ItemCount())},
		{Label: "Size", Value: render.FormatSize(table.SizeBytes())},
	}

	// Key schema
	keySchema := table.KeySchema()
	if len(keySchema) > 0 {
		var keys []string
		for _, k := range keySchema {
			keyType := "HASH"
			if k.KeyType == types.KeyTypeRange {
				keyType = "RANGE"
			}
			keys = append(keys, fmt.Sprintf("%s (%s)", *k.AttributeName, keyType))
		}
		fields = append(fields, render.SummaryField{Label: "Key Schema", Value: strings.Join(keys, ", ")})
	}

	// Capacity
	if table.BillingMode() == "PROVISIONED" {
		fields = append(fields, render.SummaryField{
			Label: "Read Capacity",
			Value: fmt.Sprintf("%d RCU", table.ReadCapacity()),
		})
		fields = append(fields, render.SummaryField{
			Label: "Write Capacity",
			Value: fmt.Sprintf("%d WCU", table.WriteCapacity()),
		})
	}

	// Indexes
	if gsiCount := table.GSICount(); gsiCount > 0 {
		var gsiNames []string
		for _, gsi := range table.GlobalSecondaryIndexes() {
			if gsi.IndexName != nil {
				gsiNames = append(gsiNames, *gsi.IndexName)
			}
		}
		fields = append(fields, render.SummaryField{
			Label: "GSIs",
			Value: fmt.Sprintf("%d: %s", gsiCount, strings.Join(gsiNames, ", ")),
		})
	}

	if lsiCount := table.LSICount(); lsiCount > 0 {
		fields = append(fields, render.SummaryField{
			Label: "LSIs",
			Value: fmt.Sprintf("%d", lsiCount),
		})
	}

	if created := table.CreationDateTime(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}
