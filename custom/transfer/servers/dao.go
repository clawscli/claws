package servers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ServerDAO provides data access for Transfer Family servers.
type ServerDAO struct {
	dao.BaseDAO
	client *transfer.Client
}

// NewServerDAO creates a new ServerDAO.
func NewServerDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ServerDAO{
		BaseDAO: dao.NewBaseDAO("transfer", "servers"),
		client:  transfer.NewFromConfig(cfg),
	}, nil
}

// List returns all Transfer Family servers.
func (d *ServerDAO) List(ctx context.Context) ([]dao.Resource, error) {
	servers, err := appaws.Paginate(ctx, func(token *string) ([]types.ListedServer, *string, error) {
		output, err := d.client.ListServers(ctx, &transfer.ListServersInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list transfer servers")
		}
		return output.Servers, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(servers))
	for i, srv := range servers {
		resources[i] = NewServerResource(srv)
	}
	return resources, nil
}

// Get returns a specific Transfer Family server by ID.
func (d *ServerDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeServer(ctx, &transfer.DescribeServerInput{
		ServerId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe transfer server %s", id)
	}
	return NewServerResourceFromDetail(*output.Server), nil
}

// Delete deletes a Transfer Family server by ID.
func (d *ServerDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteServer(ctx, &transfer.DeleteServerInput{
		ServerId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete transfer server %s", id)
	}
	return nil
}

// ServerResource wraps a Transfer Family server.
type ServerResource struct {
	dao.BaseResource
	Summary *types.ListedServer
	Detail  *types.DescribedServer
}

// NewServerResource creates a new ServerResource from summary.
func NewServerResource(srv types.ListedServer) *ServerResource {
	return &ServerResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(srv.ServerId),
			ARN: appaws.Str(srv.Arn),
		},
		Summary: &srv,
	}
}

// NewServerResourceFromDetail creates a new ServerResource from detail.
func NewServerResourceFromDetail(srv types.DescribedServer) *ServerResource {
	return &ServerResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(srv.ServerId),
			ARN: appaws.Str(srv.Arn),
		},
		Detail: &srv,
	}
}

// ServerId returns the server ID.
func (r *ServerResource) ServerId() string {
	return r.ID
}

// State returns the server state.
func (r *ServerResource) State() string {
	if r.Summary != nil {
		return string(r.Summary.State)
	}
	if r.Detail != nil {
		return string(r.Detail.State)
	}
	return ""
}

// EndpointType returns the endpoint type.
func (r *ServerResource) EndpointType() string {
	if r.Summary != nil {
		return string(r.Summary.EndpointType)
	}
	if r.Detail != nil {
		return string(r.Detail.EndpointType)
	}
	return ""
}

// Domain returns the domain.
func (r *ServerResource) Domain() string {
	if r.Summary != nil {
		return string(r.Summary.Domain)
	}
	if r.Detail != nil {
		return string(r.Detail.Domain)
	}
	return ""
}

// IdentityProviderType returns the identity provider type.
func (r *ServerResource) IdentityProviderType() string {
	if r.Summary != nil {
		return string(r.Summary.IdentityProviderType)
	}
	if r.Detail != nil {
		return string(r.Detail.IdentityProviderType)
	}
	return ""
}

// LoggingRole returns the logging role ARN.
func (r *ServerResource) LoggingRole() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.LoggingRole)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.LoggingRole)
	}
	return ""
}

// UserCount returns the user count.
func (r *ServerResource) UserCount() int32 {
	if r.Summary != nil && r.Summary.UserCount != nil {
		return *r.Summary.UserCount
	}
	return 0
}

// Protocols returns the enabled protocols.
func (r *ServerResource) Protocols() []string {
	var protocols []types.Protocol
	if r.Detail != nil {
		protocols = r.Detail.Protocols
	}
	result := make([]string, len(protocols))
	for i, p := range protocols {
		result[i] = string(p)
	}
	return result
}

// ProtocolsString returns the protocols as a comma-separated string.
func (r *ServerResource) ProtocolsString() string {
	protocols := r.Protocols()
	if len(protocols) == 0 {
		return ""
	}
	result := protocols[0]
	for i := 1; i < len(protocols); i++ {
		result += ", " + protocols[i]
	}
	return result
}

// HostKeyFingerprint returns the host key fingerprint.
func (r *ServerResource) HostKeyFingerprint() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.HostKeyFingerprint)
	}
	return ""
}

// SecurityPolicyName returns the security policy name.
func (r *ServerResource) SecurityPolicyName() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.SecurityPolicyName)
	}
	return ""
}

// Certificate returns the certificate ARN.
func (r *ServerResource) Certificate() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.Certificate)
	}
	return ""
}

// Endpoint returns the endpoint.
func (r *ServerResource) Endpoint() string {
	if r.Detail != nil && r.Detail.EndpointDetails != nil {
		// For VPC endpoints, return VPC endpoint ID
		if r.Detail.EndpointDetails.VpcEndpointId != nil {
			return appaws.Str(r.Detail.EndpointDetails.VpcEndpointId)
		}
	}
	return ""
}

// VpcId returns the VPC ID.
func (r *ServerResource) VpcId() string {
	if r.Detail != nil && r.Detail.EndpointDetails != nil {
		return appaws.Str(r.Detail.EndpointDetails.VpcId)
	}
	return ""
}

// SubnetIds returns the subnet IDs.
func (r *ServerResource) SubnetIds() []string {
	if r.Detail != nil && r.Detail.EndpointDetails != nil {
		return r.Detail.EndpointDetails.SubnetIds
	}
	return nil
}

// AddressAllocationIds returns the address allocation IDs.
func (r *ServerResource) AddressAllocationIds() []string {
	if r.Detail != nil && r.Detail.EndpointDetails != nil {
		return r.Detail.EndpointDetails.AddressAllocationIds
	}
	return nil
}

// SecurityGroupIds returns the security group IDs.
func (r *ServerResource) SecurityGroupIds() []string {
	if r.Detail != nil && r.Detail.EndpointDetails != nil {
		return r.Detail.EndpointDetails.SecurityGroupIds
	}
	return nil
}

// PreAuthenticationLoginBanner returns the pre-authentication login banner.
func (r *ServerResource) PreAuthenticationLoginBanner() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.PreAuthenticationLoginBanner)
	}
	return ""
}

// PostAuthenticationLoginBanner returns the post-authentication login banner.
func (r *ServerResource) PostAuthenticationLoginBanner() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.PostAuthenticationLoginBanner)
	}
	return ""
}

// StructuredLogDestinations returns the structured log destinations.
func (r *ServerResource) StructuredLogDestinations() []string {
	if r.Detail != nil {
		return r.Detail.StructuredLogDestinations
	}
	return nil
}

// S3StorageOptions returns the S3 storage options.
func (r *ServerResource) S3StorageOptions() string {
	if r.Detail != nil && r.Detail.S3StorageOptions != nil {
		return string(r.Detail.S3StorageOptions.DirectoryListingOptimization)
	}
	return ""
}

// Tags returns the server tags.
func (r *ServerResource) Tags() []types.Tag {
	if r.Detail != nil {
		return r.Detail.Tags
	}
	return nil
}

// IdentityProviderDetails returns identity provider configuration.
func (r *ServerResource) IdentityProviderDetails() *types.IdentityProviderDetails {
	if r.Detail != nil {
		return r.Detail.IdentityProviderDetails
	}
	return nil
}

// ProtocolDetails returns protocol configuration.
func (r *ServerResource) ProtocolDetails() *types.ProtocolDetails {
	if r.Detail != nil {
		return r.Detail.ProtocolDetails
	}
	return nil
}

// WorkflowDetails returns workflow configuration.
func (r *ServerResource) WorkflowDetails() *types.WorkflowDetails {
	if r.Detail != nil {
		return r.Detail.WorkflowDetails
	}
	return nil
}

// As2ServiceManagedEgressIpAddresses returns the AS2 service managed egress IP addresses.
func (r *ServerResource) As2ServiceManagedEgressIpAddresses() []string {
	if r.Detail != nil {
		return r.Detail.As2ServiceManagedEgressIpAddresses
	}
	return nil
}
