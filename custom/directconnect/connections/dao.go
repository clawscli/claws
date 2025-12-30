package connections

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ConnectionDAO provides data access for Direct Connect connections.
type ConnectionDAO struct {
	dao.BaseDAO
	client *directconnect.Client
}

// NewConnectionDAO creates a new ConnectionDAO.
func NewConnectionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new directconnect/connections dao: %w", err)
	}
	return &ConnectionDAO{
		BaseDAO: dao.NewBaseDAO("directconnect", "connections"),
		client:  directconnect.NewFromConfig(cfg),
	}, nil
}

// List returns all Direct Connect connections.
func (d *ConnectionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.DescribeConnections(ctx, &directconnect.DescribeConnectionsInput{})
	if err != nil {
		return nil, fmt.Errorf("describe direct connect connections: %w", err)
	}

	resources := make([]dao.Resource, len(output.Connections))
	for i, conn := range output.Connections {
		resources[i] = NewConnectionResource(conn)
	}
	return resources, nil
}

// Get returns a specific connection by ID.
func (d *ConnectionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeConnections(ctx, &directconnect.DescribeConnectionsInput{
		ConnectionId: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("describe direct connect connection %s: %w", id, err)
	}
	if len(output.Connections) == 0 {
		return nil, fmt.Errorf("direct connect connection not found: %s", id)
	}
	return NewConnectionResource(output.Connections[0]), nil
}

// Delete deletes a connection by ID.
func (d *ConnectionDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteConnection(ctx, &directconnect.DeleteConnectionInput{
		ConnectionId: &id,
	})
	if err != nil {
		return fmt.Errorf("delete direct connect connection %s: %w", id, err)
	}
	return nil
}

// ConnectionResource wraps a Direct Connect connection.
type ConnectionResource struct {
	dao.BaseResource
	Item types.Connection
}

// NewConnectionResource creates a new ConnectionResource.
func NewConnectionResource(conn types.Connection) *ConnectionResource {
	return &ConnectionResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(conn.ConnectionId),
			ARN: "",
		},
		Item: conn,
	}
}

// ConnectionName returns the connection name.
func (r *ConnectionResource) ConnectionName() string {
	return appaws.Str(r.Item.ConnectionName)
}

// ConnectionState returns the connection state.
func (r *ConnectionResource) ConnectionState() string {
	return string(r.Item.ConnectionState)
}

// Location returns the location.
func (r *ConnectionResource) Location() string {
	return appaws.Str(r.Item.Location)
}

// Bandwidth returns the bandwidth.
func (r *ConnectionResource) Bandwidth() string {
	return appaws.Str(r.Item.Bandwidth)
}

// Vlan returns the VLAN.
func (r *ConnectionResource) Vlan() int32 {
	return r.Item.Vlan
}

// PartnerName returns the partner name.
func (r *ConnectionResource) PartnerName() string {
	return appaws.Str(r.Item.PartnerName)
}

// OwnerAccount returns the owner account.
func (r *ConnectionResource) OwnerAccount() string {
	return appaws.Str(r.Item.OwnerAccount)
}

// Region returns the region.
func (r *ConnectionResource) Region() string {
	return appaws.Str(r.Item.Region)
}

// HasLogicalRedundancy returns whether it has logical redundancy.
func (r *ConnectionResource) HasLogicalRedundancy() string {
	return string(r.Item.HasLogicalRedundancy)
}

// EncryptionMode returns the encryption mode.
func (r *ConnectionResource) EncryptionMode() string {
	return appaws.Str(r.Item.EncryptionMode)
}

// AwsDeviceV2 returns the AWS device V2.
func (r *ConnectionResource) AwsDeviceV2() string {
	return appaws.Str(r.Item.AwsDeviceV2)
}

// AwsLogicalDeviceId returns the AWS logical device ID.
func (r *ConnectionResource) AwsLogicalDeviceId() string {
	return appaws.Str(r.Item.AwsLogicalDeviceId)
}

// JumboFrameCapable returns if jumbo frames are capable.
func (r *ConnectionResource) JumboFrameCapable() bool {
	return appaws.Bool(r.Item.JumboFrameCapable)
}

// MacSecCapable returns if MACSec is capable.
func (r *ConnectionResource) MacSecCapable() bool {
	return appaws.Bool(r.Item.MacSecCapable)
}

// PortEncryptionStatus returns the port encryption status.
func (r *ConnectionResource) PortEncryptionStatus() string {
	return appaws.Str(r.Item.PortEncryptionStatus)
}

// ProviderName returns the provider name.
func (r *ConnectionResource) ProviderName() string {
	return appaws.Str(r.Item.ProviderName)
}

// LagId returns the LAG ID.
func (r *ConnectionResource) LagId() string {
	return appaws.Str(r.Item.LagId)
}

// Tags returns all tags.
func (r *ConnectionResource) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range r.Item.Tags {
		tags[appaws.Str(tag.Key)] = appaws.Str(tag.Value)
	}
	return tags
}
