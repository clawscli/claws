package hostedzones

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

// HostedZoneDAO provides data access for Route53 hosted zones
type HostedZoneDAO struct {
	dao.BaseDAO
	client *route53.Client
}

// NewHostedZoneDAO creates a new HostedZoneDAO
func NewHostedZoneDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new route53/hostedzones dao")
	}
	return &HostedZoneDAO{
		BaseDAO: dao.NewBaseDAO("route53", "hosted-zones"),
		client:  route53.NewFromConfig(cfg),
	}, nil
}

// List returns all hosted zones
func (d *HostedZoneDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &route53.ListHostedZonesInput{}

	var resources []dao.Resource
	paginator := route53.NewListHostedZonesPaginator(d.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list hosted zones")
		}

		for _, zone := range output.HostedZones {
			resources = append(resources, NewHostedZoneResource(zone))
		}
	}

	return resources, nil
}

// Get returns a specific hosted zone by ID
func (d *HostedZoneDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// ID might be just the zone ID or full path like /hostedzone/XXXXX
	zoneID := id
	if !strings.HasPrefix(id, "/hostedzone/") {
		zoneID = "/hostedzone/" + id
	}

	output, err := d.client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
		Id: &zoneID,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get hosted zone %s", id)
	}

	return NewHostedZoneResourceWithDetails(output), nil
}

// Delete is not supported for hosted zones (requires empty zone first)
func (d *HostedZoneDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for hosted zones")
}

// HostedZoneResource wraps a Route53 hosted zone
type HostedZoneResource struct {
	dao.BaseResource
	Item           types.HostedZone
	DelegationSet  *types.DelegationSet
	VPCs           []types.VPC
	RecordSetCount int64
}

// NewHostedZoneResource creates a new HostedZoneResource
func NewHostedZoneResource(zone types.HostedZone) *HostedZoneResource {
	id := ""
	if zone.Id != nil {
		// Remove /hostedzone/ prefix if present
		id = strings.TrimPrefix(*zone.Id, "/hostedzone/")
	}

	name := ""
	if zone.Name != nil {
		name = strings.TrimSuffix(*zone.Name, ".")
	}

	return &HostedZoneResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Tags: nil,
			Data: zone,
		},
		Item:           zone,
		RecordSetCount: appaws.Int64(zone.ResourceRecordSetCount),
	}
}

// NewHostedZoneResourceWithDetails creates a HostedZoneResource with full details
func NewHostedZoneResourceWithDetails(output *route53.GetHostedZoneOutput) *HostedZoneResource {
	if output.HostedZone == nil {
		return nil
	}

	r := NewHostedZoneResource(*output.HostedZone)
	r.DelegationSet = output.DelegationSet
	r.VPCs = output.VPCs
	return r
}

// ZoneID returns the hosted zone ID (without /hostedzone/ prefix)
func (r *HostedZoneResource) ZoneID() string {
	if r.Item.Id != nil {
		return strings.TrimPrefix(*r.Item.Id, "/hostedzone/")
	}
	return ""
}

// DomainName returns the domain name (without trailing dot)
func (r *HostedZoneResource) DomainName() string {
	if r.Item.Name != nil {
		return strings.TrimSuffix(*r.Item.Name, ".")
	}
	return ""
}

// Comment returns the zone comment
func (r *HostedZoneResource) Comment() string {
	if r.Item.Config != nil && r.Item.Config.Comment != nil {
		return *r.Item.Config.Comment
	}
	return ""
}

// IsPrivate returns whether this is a private hosted zone
func (r *HostedZoneResource) IsPrivate() bool {
	if r.Item.Config != nil {
		return r.Item.Config.PrivateZone
	}
	return false
}

// CallerReference returns the caller reference
func (r *HostedZoneResource) CallerReference() string {
	if r.Item.CallerReference != nil {
		return *r.Item.CallerReference
	}
	return ""
}

// NameServers returns the name servers for public zones
func (r *HostedZoneResource) NameServers() []string {
	if r.DelegationSet != nil {
		return r.DelegationSet.NameServers
	}
	return nil
}
