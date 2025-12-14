package queues

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// QueueRenderer renders SQS queues
type QueueRenderer struct {
	render.BaseRenderer
}

// NewQueueRenderer creates a new QueueRenderer
func NewQueueRenderer() render.Renderer {
	return &QueueRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sqs",
			Resource: "queues",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetName() }, Priority: 0},
				{Name: "TYPE", Width: 10, Getter: getType, Priority: 1},
				{Name: "MESSAGES", Width: 10, Getter: getMessages, Priority: 2},
				{Name: "IN FLIGHT", Width: 10, Getter: getInFlight, Priority: 3},
				{Name: "DELAYED", Width: 10, Getter: getDelayed, Priority: 4},
				{Name: "RETENTION", Width: 10, Getter: getRetention, Priority: 5},
			},
		},
	}
}

func getType(r dao.Resource) string {
	if q, ok := r.(*QueueResource); ok {
		if q.IsFIFO() {
			return "FIFO"
		}
		return "Standard"
	}
	return ""
}

func getMessages(r dao.Resource) string {
	if q, ok := r.(*QueueResource); ok {
		return q.ApproximateNumberOfMessages()
	}
	return ""
}

func getInFlight(r dao.Resource) string {
	if q, ok := r.(*QueueResource); ok {
		count := q.ApproximateNumberOfMessagesNotVisible()
		if count == "0" {
			return "-"
		}
		return count
	}
	return ""
}

func getDelayed(r dao.Resource) string {
	if q, ok := r.(*QueueResource); ok {
		count := q.ApproximateNumberOfMessagesDelayed()
		if count == "0" {
			return "-"
		}
		return count
	}
	return ""
}

func getRetention(r dao.Resource) string {
	if q, ok := r.(*QueueResource); ok {
		seconds := q.MessageRetentionPeriod()
		if seconds == "" {
			return ""
		}
		secs, err := strconv.Atoi(seconds)
		if err != nil {
			return seconds
		}
		days := secs / 86400
		if days > 0 {
			return fmt.Sprintf("%dd", days)
		}
		hours := secs / 3600
		if hours > 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%ds", secs)
	}
	return ""
}

// RenderDetail renders detailed queue information
func (r *QueueRenderer) RenderDetail(resource dao.Resource) string {
	q, ok := resource.(*QueueResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	qType := "Standard"
	if q.IsFIFO() {
		qType = "FIFO"
	}

	d.Title("SQS Queue", q.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", q.GetName())
	d.Field("ARN", q.GetARN())
	d.Field("URL", q.URL)
	d.Field("Type", qType)

	// Messages
	d.Section("Messages")
	d.Field("Available", q.ApproximateNumberOfMessages())
	d.Field("In Flight", q.ApproximateNumberOfMessagesNotVisible())
	d.Field("Delayed", q.ApproximateNumberOfMessagesDelayed())

	// Configuration
	d.Section("Configuration")
	if vt := q.VisibilityTimeout(); vt != "" {
		d.Field("Visibility Timeout", vt+" seconds")
	}
	if ret := q.MessageRetentionPeriod(); ret != "" {
		secs, _ := strconv.Atoi(ret)
		days := secs / 86400
		d.Field("Message Retention", fmt.Sprintf("%d days", days))
	}
	if delay := q.DelaySeconds(); delay != "" {
		d.Field("Delivery Delay", delay+" seconds")
	}
	if wait := q.ReceiveMessageWaitTimeSeconds(); wait != "" {
		d.Field("Long Polling Wait", wait+" seconds")
	}
	if maxSize := q.Attributes["MaximumMessageSize"]; maxSize != "" {
		sizeBytes, _ := strconv.Atoi(maxSize)
		d.Field("Max Message Size", fmt.Sprintf("%d KB", sizeBytes/1024))
	}

	// Encryption
	d.Section("Encryption")
	if q.Attributes["SqsManagedSseEnabled"] == "true" {
		d.Field("Server-Side Encryption", "SQS-managed (SSE-SQS)")
	} else if kmsKey := q.Attributes["KmsMasterKeyId"]; kmsKey != "" {
		d.Field("Server-Side Encryption", "KMS")
		d.Field("KMS Key ID", kmsKey)
		if reuseSeconds := q.Attributes["KmsDataKeyReusePeriodSeconds"]; reuseSeconds != "" {
			d.Field("Key Reuse Period", reuseSeconds+" seconds")
		}
	} else {
		d.Field("Server-Side Encryption", "Disabled")
	}

	// FIFO-specific settings
	if q.IsFIFO() {
		d.Section("FIFO Settings")
		if q.Attributes["ContentBasedDeduplication"] == "true" {
			d.Field("Content-Based Deduplication", "Enabled")
		} else {
			d.Field("Content-Based Deduplication", "Disabled")
		}
		if scope := q.Attributes["DeduplicationScope"]; scope != "" {
			d.Field("Deduplication Scope", scope)
		}
		if limit := q.Attributes["FifoThroughputLimit"]; limit != "" {
			d.Field("FIFO Throughput Limit", limit)
		}
	}
	// Dead Letter Queue
	if redrive := q.RedrivePolicy(); redrive != "" {
		var policy struct {
			DeadLetterTargetArn string `json:"deadLetterTargetArn"`
			MaxReceiveCount     int    `json:"maxReceiveCount"`
		}
		if err := json.Unmarshal([]byte(redrive), &policy); err == nil {
			d.Section("Dead Letter Queue")
			parts := strings.Split(policy.DeadLetterTargetArn, ":")
			d.Field("Target Queue", parts[len(parts)-1])
			d.Field("Max Receives", fmt.Sprintf("%d", policy.MaxReceiveCount))
		}
	}

	// Timestamps
	if created := q.CreatedTimestamp(); created != "" {
		ts, err := strconv.ParseInt(created, 10, 64)
		if err == nil {
			t := time.Unix(ts, 0)
			d.Section("Timestamps")
			d.Field("Created", t.Format("2006-01-02 15:04:05"))
		}
	}
	if modified := q.LastModifiedTimestamp(); modified != "" {
		ts, err := strconv.ParseInt(modified, 10, 64)
		if err == nil {
			t := time.Unix(ts, 0)
			d.Field("Last Modified", t.Format("2006-01-02 15:04:05"))
		}
	}

	// Tags
	d.Tags(q.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *QueueRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	q, ok := resource.(*QueueResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	qType := "Standard"
	if q.IsFIFO() {
		qType = "FIFO"
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: q.GetName()},
		{Label: "ARN", Value: q.GetARN()},
		{Label: "URL", Value: q.URL},
		{Label: "Type", Value: qType},
		{Label: "Messages Available", Value: q.ApproximateNumberOfMessages()},
		{Label: "Messages In Flight", Value: q.ApproximateNumberOfMessagesNotVisible()},
		{Label: "Messages Delayed", Value: q.ApproximateNumberOfMessagesDelayed()},
	}

	// Visibility Timeout
	if vt := q.VisibilityTimeout(); vt != "" {
		fields = append(fields, render.SummaryField{
			Label: "Visibility Timeout",
			Value: vt + "s",
		})
	}

	// Message Retention
	if ret := q.MessageRetentionPeriod(); ret != "" {
		secs, _ := strconv.Atoi(ret)
		days := secs / 86400
		fields = append(fields, render.SummaryField{
			Label: "Message Retention",
			Value: fmt.Sprintf("%d days", days),
		})
	}

	// Delay
	if delay := q.DelaySeconds(); delay != "" && delay != "0" {
		fields = append(fields, render.SummaryField{
			Label: "Delivery Delay",
			Value: delay + "s",
		})
	}

	// Long Polling
	if wait := q.ReceiveMessageWaitTimeSeconds(); wait != "" && wait != "0" {
		fields = append(fields, render.SummaryField{
			Label: "Long Polling Wait",
			Value: wait + "s",
		})
	}

	// Created
	if created := q.CreatedTimestamp(); created != "" {
		ts, err := strconv.ParseInt(created, 10, 64)
		if err == nil {
			t := time.Unix(ts, 0)
			fields = append(fields, render.SummaryField{
				Label: "Created",
				Value: t.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// DLQ
	if redrive := q.RedrivePolicy(); redrive != "" {
		var policy struct {
			DeadLetterTargetArn string `json:"deadLetterTargetArn"`
			MaxReceiveCount     int    `json:"maxReceiveCount"`
		}
		if err := json.Unmarshal([]byte(redrive), &policy); err == nil {
			// Extract queue name from ARN
			parts := strings.Split(policy.DeadLetterTargetArn, ":")
			dlqName := parts[len(parts)-1]
			fields = append(fields, render.SummaryField{
				Label: "Dead Letter Queue",
				Value: fmt.Sprintf("%s (max %d receives)", dlqName, policy.MaxReceiveCount),
			})
		}
	}

	return fields
}
