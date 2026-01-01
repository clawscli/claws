package recordsets

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RecordSetDAO provides data access for Route53 record sets
type RecordSetDAO struct {
	dao.BaseDAO
	client *route53.Client
}

// NewRecordSetDAO creates a new RecordSetDAO
func NewRecordSetDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new route53/recordsets dao")
	}
	return &RecordSetDAO{
		BaseDAO: dao.NewBaseDAO("route53", "record-sets"),
		client:  route53.NewFromConfig(cfg),
	}, nil
}

// List returns record sets (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *RecordSetDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 300, "")
	return resources, err
}

// ListPage returns a page of Route53 record sets.
// Implements dao.PaginatedDAO interface.
// Note: pageToken format is "name:type" for Route53 pagination.
func (d *RecordSetDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get filter from context
	hostedZoneID := dao.GetFilterFromContext(ctx, "HostedZoneId")
	if hostedZoneID == "" {
		return nil, "", fmt.Errorf("HostedZoneId filter required - navigate from a hosted zone")
	}

	maxItems := int32(pageSize)
	if maxItems > 300 {
		maxItems = 300 // AWS API max
	}

	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: &hostedZoneID,
		MaxItems:     &maxItems,
	}

	// Parse pageToken (format: "name:type")
	if pageToken != "" {
		parts := strings.SplitN(pageToken, ":", 2)
		if len(parts) == 2 {
			input.StartRecordName = &parts[0]
			input.StartRecordType = types.RRType(parts[1])
		}
	}

	output, err := d.client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list record sets")
	}

	resources := make([]dao.Resource, 0, len(output.ResourceRecordSets))
	for _, recordSet := range output.ResourceRecordSets {
		resources = append(resources, NewRecordSetResource(recordSet, hostedZoneID))
	}

	// Build next token from IsTruncated and NextRecordName/NextRecordType
	nextToken := ""
	if output.IsTruncated && output.NextRecordName != nil {
		nextToken = fmt.Sprintf("%s:%s", *output.NextRecordName, output.NextRecordType)
	}

	return resources, nextToken, nil
}

// Get returns a specific record set (not directly supported by Route53 API)
func (d *RecordSetDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// ID format: hostedZoneId:name:type
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid record set ID format: %s", id)
	}

	hostedZoneID := parts[0]
	name := parts[1]
	recordType := parts[2]

	// List and find the specific record
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    &hostedZoneID,
		StartRecordName: &name,
		StartRecordType: types.RRType(recordType),
		MaxItems:        intPtr(1),
	}

	output, err := d.client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get record set %s", id)
	}

	for _, rs := range output.ResourceRecordSets {
		if rs.Name != nil && rs.Type != "" {
			rsName := strings.TrimSuffix(*rs.Name, ".")
			if rsName == strings.TrimSuffix(name, ".") && string(rs.Type) == recordType {
				return NewRecordSetResource(rs, hostedZoneID), nil
			}
		}
	}

	return nil, fmt.Errorf("record set not found: %s", id)
}

// Delete is not implemented (requires ChangeResourceRecordSets)
func (d *RecordSetDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for record sets via this interface")
}

func intPtr(i int32) *int32 {
	return &i
}

// RecordSetResource wraps a Route53 resource record set
type RecordSetResource struct {
	dao.BaseResource
	Item         types.ResourceRecordSet
	HostedZoneID string
}

// NewRecordSetResource creates a new RecordSetResource
func NewRecordSetResource(rs types.ResourceRecordSet, hostedZoneID string) *RecordSetResource {
	name := ""
	if rs.Name != nil {
		name = strings.TrimSuffix(*rs.Name, ".")
	}

	// Create unique ID: hostedZoneId:name:type
	id := fmt.Sprintf("%s:%s:%s", hostedZoneID, name, string(rs.Type))

	return &RecordSetResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Tags: nil,
			Data: rs,
		},
		Item:         rs,
		HostedZoneID: hostedZoneID,
	}
}

// RecordName returns the record name (without trailing dot)
func (r *RecordSetResource) RecordName() string {
	if r.Item.Name != nil {
		return strings.TrimSuffix(*r.Item.Name, ".")
	}
	return ""
}

// RecordType returns the record type (A, AAAA, CNAME, etc.)
func (r *RecordSetResource) RecordType() string {
	return string(r.Item.Type)
}

// TTL returns the TTL value
func (r *RecordSetResource) TTL() int64 {
	if r.Item.TTL != nil {
		return *r.Item.TTL
	}
	return 0
}

// Values returns the record values
func (r *RecordSetResource) Values() []string {
	var values []string
	for _, rr := range r.Item.ResourceRecords {
		if rr.Value != nil {
			values = append(values, *rr.Value)
		}
	}
	return values
}

// IsAlias returns whether this is an alias record
func (r *RecordSetResource) IsAlias() bool {
	return r.Item.AliasTarget != nil
}

// AliasTarget returns the alias target DNS name
func (r *RecordSetResource) AliasTarget() string {
	if r.Item.AliasTarget != nil && r.Item.AliasTarget.DNSName != nil {
		return strings.TrimSuffix(*r.Item.AliasTarget.DNSName, ".")
	}
	return ""
}

// AliasHostedZoneID returns the alias target hosted zone ID
func (r *RecordSetResource) AliasHostedZoneID() string {
	if r.Item.AliasTarget != nil && r.Item.AliasTarget.HostedZoneId != nil {
		return *r.Item.AliasTarget.HostedZoneId
	}
	return ""
}

// SetIdentifier returns the set identifier for weighted/latency/failover records
func (r *RecordSetResource) SetIdentifier() string {
	if r.Item.SetIdentifier != nil {
		return *r.Item.SetIdentifier
	}
	return ""
}

// Weight returns the weight for weighted routing
func (r *RecordSetResource) Weight() int64 {
	if r.Item.Weight != nil {
		return *r.Item.Weight
	}
	return 0
}

// Region returns the region for latency-based routing
func (r *RecordSetResource) Region() string {
	return string(r.Item.Region)
}

// Failover returns the failover type (PRIMARY or SECONDARY)
func (r *RecordSetResource) Failover() string {
	return string(r.Item.Failover)
}

// HealthCheckID returns the associated health check ID
func (r *RecordSetResource) HealthCheckID() string {
	if r.Item.HealthCheckId != nil {
		return *r.Item.HealthCheckId
	}
	return ""
}
