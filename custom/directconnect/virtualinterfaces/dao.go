package virtualinterfaces

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// VirtualInterfaceDAO provides data access for Direct Connect virtual interfaces.
type VirtualInterfaceDAO struct {
	dao.BaseDAO
	client *directconnect.Client
}

// NewVirtualInterfaceDAO creates a new VirtualInterfaceDAO.
func NewVirtualInterfaceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new directconnect/virtualinterfaces dao")
	}
	return &VirtualInterfaceDAO{
		BaseDAO: dao.NewBaseDAO("directconnect", "virtual-interfaces"),
		client:  directconnect.NewFromConfig(cfg),
	}, nil
}

// List returns all Direct Connect virtual interfaces, optionally filtered by connection ID.
func (d *VirtualInterfaceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &directconnect.DescribeVirtualInterfacesInput{}

	// Filter by Connection ID if provided
	if connID := dao.GetFilterFromContext(ctx, "ConnectionId"); connID != "" {
		input.ConnectionId = &connID
	}

	output, err := d.client.DescribeVirtualInterfaces(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "describe virtual interfaces")
	}

	resources := make([]dao.Resource, len(output.VirtualInterfaces))
	for i, vi := range output.VirtualInterfaces {
		resources[i] = NewVirtualInterfaceResource(vi)
	}
	return resources, nil
}

// Get returns a specific virtual interface by ID.
func (d *VirtualInterfaceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeVirtualInterfaces(ctx, &directconnect.DescribeVirtualInterfacesInput{
		VirtualInterfaceId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe virtual interface %s", id)
	}
	if len(output.VirtualInterfaces) == 0 {
		return nil, fmt.Errorf("virtual interface not found: %s", id)
	}
	return NewVirtualInterfaceResource(output.VirtualInterfaces[0]), nil
}

// Delete deletes a virtual interface by ID.
func (d *VirtualInterfaceDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteVirtualInterface(ctx, &directconnect.DeleteVirtualInterfaceInput{
		VirtualInterfaceId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete virtual interface %s", id)
	}
	return nil
}

// VirtualInterfaceResource wraps a Direct Connect virtual interface.
type VirtualInterfaceResource struct {
	dao.BaseResource
	Item types.VirtualInterface
}

// NewVirtualInterfaceResource creates a new VirtualInterfaceResource.
func NewVirtualInterfaceResource(vi types.VirtualInterface) *VirtualInterfaceResource {
	return &VirtualInterfaceResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(vi.VirtualInterfaceId),
			ARN: "",
		},
		Item: vi,
	}
}

// VirtualInterfaceName returns the virtual interface name.
func (r *VirtualInterfaceResource) VirtualInterfaceName() string {
	return appaws.Str(r.Item.VirtualInterfaceName)
}

// VirtualInterfaceState returns the state.
func (r *VirtualInterfaceResource) VirtualInterfaceState() string {
	return string(r.Item.VirtualInterfaceState)
}

// VirtualInterfaceType returns the type (private, public, transit).
func (r *VirtualInterfaceResource) VirtualInterfaceType() string {
	return appaws.Str(r.Item.VirtualInterfaceType)
}

// ConnectionId returns the connection ID.
func (r *VirtualInterfaceResource) ConnectionId() string {
	return appaws.Str(r.Item.ConnectionId)
}

// Vlan returns the VLAN.
func (r *VirtualInterfaceResource) Vlan() int32 {
	return r.Item.Vlan
}

// Asn returns the ASN.
func (r *VirtualInterfaceResource) Asn() int32 {
	return r.Item.Asn
}

// AmazonSideAsn returns the Amazon side ASN.
func (r *VirtualInterfaceResource) AmazonSideAsn() int64 {
	return appaws.Int64(r.Item.AmazonSideAsn)
}

// AmazonAddress returns the Amazon address.
func (r *VirtualInterfaceResource) AmazonAddress() string {
	return appaws.Str(r.Item.AmazonAddress)
}

// CustomerAddress returns the customer address.
func (r *VirtualInterfaceResource) CustomerAddress() string {
	return appaws.Str(r.Item.CustomerAddress)
}

// Location returns the location.
func (r *VirtualInterfaceResource) Location() string {
	return appaws.Str(r.Item.Location)
}

// Region returns the region.
func (r *VirtualInterfaceResource) Region() string {
	return appaws.Str(r.Item.Region)
}

// OwnerAccount returns the owner account.
func (r *VirtualInterfaceResource) OwnerAccount() string {
	return appaws.Str(r.Item.OwnerAccount)
}

// VirtualGatewayId returns the virtual gateway ID.
func (r *VirtualInterfaceResource) VirtualGatewayId() string {
	return appaws.Str(r.Item.VirtualGatewayId)
}

// DirectConnectGatewayId returns the Direct Connect gateway ID.
func (r *VirtualInterfaceResource) DirectConnectGatewayId() string {
	return appaws.Str(r.Item.DirectConnectGatewayId)
}

// Mtu returns the MTU.
func (r *VirtualInterfaceResource) Mtu() int32 {
	if r.Item.Mtu != nil {
		return *r.Item.Mtu
	}
	return 0
}

// JumboFrameCapable returns whether jumbo frames are supported.
func (r *VirtualInterfaceResource) JumboFrameCapable() bool {
	if r.Item.JumboFrameCapable != nil {
		return *r.Item.JumboFrameCapable
	}
	return false
}

// AuthKey returns the authentication key.
func (r *VirtualInterfaceResource) AuthKey() string {
	return appaws.Str(r.Item.AuthKey)
}

// AddressFamily returns the address family.
func (r *VirtualInterfaceResource) AddressFamily() string {
	return string(r.Item.AddressFamily)
}

// BgpPeers returns the BGP peers.
func (r *VirtualInterfaceResource) BgpPeers() []types.BGPPeer {
	return r.Item.BgpPeers
}

// RouteFilterPrefixes returns the route filter prefixes.
func (r *VirtualInterfaceResource) RouteFilterPrefixes() []string {
	var prefixes []string
	for _, p := range r.Item.RouteFilterPrefixes {
		if p.Cidr != nil {
			prefixes = append(prefixes, *p.Cidr)
		}
	}
	return prefixes
}

// SiteLinkEnabled returns whether SiteLink is enabled.
func (r *VirtualInterfaceResource) SiteLinkEnabled() bool {
	return appaws.Bool(r.Item.SiteLinkEnabled)
}

// Tags returns all tags.
func (r *VirtualInterfaceResource) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range r.Item.Tags {
		tags[appaws.Str(tag.Key)] = appaws.Str(tag.Value)
	}
	return tags
}
