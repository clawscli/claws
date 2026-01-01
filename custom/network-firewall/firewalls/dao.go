package firewalls

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FirewallDAO provides data access for Network Firewalls.
type FirewallDAO struct {
	dao.BaseDAO
	client *networkfirewall.Client
}

// NewFirewallDAO creates a new FirewallDAO.
func NewFirewallDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new networkfirewall/firewalls dao")
	}
	return &FirewallDAO{
		BaseDAO: dao.NewBaseDAO("network-firewall", "firewalls"),
		client:  networkfirewall.NewFromConfig(cfg),
	}, nil
}

// List returns all Network Firewalls.
func (d *FirewallDAO) List(ctx context.Context) ([]dao.Resource, error) {
	firewalls, err := appaws.Paginate(ctx, func(token *string) ([]types.FirewallMetadata, *string, error) {
		output, err := d.client.ListFirewalls(ctx, &networkfirewall.ListFirewallsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list network firewalls")
		}
		return output.Firewalls, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(firewalls))
	for i, fw := range firewalls {
		resources[i] = NewFirewallResource(fw)
	}
	return resources, nil
}

// Get returns a specific firewall by name.
func (d *FirewallDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{
		FirewallName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe network firewall %s", id)
	}
	return NewFirewallResourceFromDetail(*output.Firewall, output.FirewallStatus), nil
}

// Delete deletes a firewall by name.
func (d *FirewallDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteFirewall(ctx, &networkfirewall.DeleteFirewallInput{
		FirewallName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete network firewall %s", id)
	}
	return nil
}

// FirewallResource wraps a Network Firewall.
type FirewallResource struct {
	dao.BaseResource
	Metadata *types.FirewallMetadata
	Detail   *types.Firewall
	Status   *types.FirewallStatus
}

// NewFirewallResource creates a new FirewallResource from metadata.
func NewFirewallResource(fw types.FirewallMetadata) *FirewallResource {
	return &FirewallResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(fw.FirewallName),
			ARN: appaws.Str(fw.FirewallArn),
		},
		Metadata: &fw,
	}
}

// NewFirewallResourceFromDetail creates a new FirewallResource from detail.
func NewFirewallResourceFromDetail(fw types.Firewall, status *types.FirewallStatus) *FirewallResource {
	return &FirewallResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(fw.FirewallName),
			ARN: appaws.Str(fw.FirewallArn),
		},
		Detail: &fw,
		Status: status,
	}
}

// FirewallName returns the firewall name.
func (r *FirewallResource) FirewallName() string {
	return r.ID
}

// VpcId returns the VPC ID.
func (r *FirewallResource) VpcId() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.VpcId)
	}
	return ""
}

// FirewallPolicyArn returns the policy ARN.
func (r *FirewallResource) FirewallPolicyArn() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.FirewallPolicyArn)
	}
	return ""
}

// Description returns the description.
func (r *FirewallResource) Description() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.Description)
	}
	return ""
}

// Status returns the firewall status.
func (r *FirewallResource) StatusValue() string {
	if r.Status != nil {
		return string(r.Status.Status)
	}
	return ""
}

// DeleteProtection returns whether delete protection is enabled.
func (r *FirewallResource) DeleteProtection() bool {
	if r.Detail != nil {
		return r.Detail.DeleteProtection
	}
	return false
}

// SubnetChangeProtection returns whether subnet change protection is enabled.
func (r *FirewallResource) SubnetChangeProtection() bool {
	if r.Detail != nil {
		return r.Detail.SubnetChangeProtection
	}
	return false
}

// PolicyChangeProtection returns whether policy change protection is enabled.
func (r *FirewallResource) PolicyChangeProtection() bool {
	if r.Detail != nil {
		return r.Detail.FirewallPolicyChangeProtection
	}
	return false
}

// SubnetMappings returns the subnet mappings.
func (r *FirewallResource) SubnetMappings() []string {
	if r.Detail == nil {
		return nil
	}
	var subnets []string
	for _, sm := range r.Detail.SubnetMappings {
		if sm.SubnetId != nil {
			subnets = append(subnets, *sm.SubnetId)
		}
	}
	return subnets
}

// SyncStates returns endpoint information per AZ.
func (r *FirewallResource) SyncStates() map[string]string {
	if r.Status == nil {
		return nil
	}
	states := make(map[string]string)
	for az, state := range r.Status.SyncStates {
		if state.Attachment != nil && state.Attachment.EndpointId != nil {
			states[az] = *state.Attachment.EndpointId
		}
	}
	return states
}

// ConfigurationSyncStateSummary returns the config sync state summary.
func (r *FirewallResource) ConfigurationSyncStateSummary() string {
	if r.Status != nil {
		return string(r.Status.ConfigurationSyncStateSummary)
	}
	return ""
}

// EncryptionConfiguration returns the encryption key ID.
func (r *FirewallResource) EncryptionKeyId() string {
	if r.Detail != nil && r.Detail.EncryptionConfiguration != nil {
		return appaws.Str(r.Detail.EncryptionConfiguration.KeyId)
	}
	return ""
}

// Tags returns all tags.
func (r *FirewallResource) Tags() map[string]string {
	if r.Detail == nil {
		return nil
	}
	tags := make(map[string]string)
	for _, tag := range r.Detail.Tags {
		tags[appaws.Str(tag.Key)] = appaws.Str(tag.Value)
	}
	return tags
}
