package domains

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/opensearch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// DomainDAO provides data access for OpenSearch domains
type DomainDAO struct {
	dao.BaseDAO
	client *opensearch.Client
}

// NewDomainDAO creates a new DomainDAO
func NewDomainDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new opensearch/domains dao: %w", err)
	}
	return &DomainDAO{
		BaseDAO: dao.NewBaseDAO("opensearch", "domains"),
		client:  opensearch.NewFromConfig(cfg),
	}, nil
}

// List returns all OpenSearch domains
func (d *DomainDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// First, get the list of domain names
	listInput := &opensearch.ListDomainNamesInput{}
	listOutput, err := d.client.ListDomainNames(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("list domain names: %w", err)
	}

	if len(listOutput.DomainNames) == 0 {
		return nil, nil
	}

	// Get domain names
	var domainNames []string
	for _, domain := range listOutput.DomainNames {
		if domain.DomainName != nil {
			domainNames = append(domainNames, *domain.DomainName)
		}
	}

	// Batch describe domains (max 5 at a time)
	var resources []dao.Resource
	for i := 0; i < len(domainNames); i += 5 {
		end := i + 5
		if end > len(domainNames) {
			end = len(domainNames)
		}
		batch := domainNames[i:end]

		describeInput := &opensearch.DescribeDomainsInput{
			DomainNames: batch,
		}
		describeOutput, err := d.client.DescribeDomains(ctx, describeInput)
		if err != nil {
			return nil, fmt.Errorf("describe domains: %w", err)
		}

		for _, status := range describeOutput.DomainStatusList {
			resources = append(resources, NewDomainResource(status))
		}
	}

	return resources, nil
}

// Get returns a specific OpenSearch domain by name
func (d *DomainDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &opensearch.DescribeDomainsInput{
		DomainNames: []string{id},
	}

	output, err := d.client.DescribeDomains(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe domain %s: %w", id, err)
	}

	if len(output.DomainStatusList) == 0 {
		return nil, fmt.Errorf("domain %s not found", id)
	}

	return NewDomainResource(output.DomainStatusList[0]), nil
}

// Delete deletes an OpenSearch domain
func (d *DomainDAO) Delete(ctx context.Context, id string) error {
	input := &opensearch.DeleteDomainInput{
		DomainName: &id,
	}

	_, err := d.client.DeleteDomain(ctx, input)
	if err != nil {
		return fmt.Errorf("delete domain %s: %w", id, err)
	}

	return nil
}

// DomainResource represents an OpenSearch domain
type DomainResource struct {
	dao.BaseResource
	Item types.DomainStatus
}

// NewDomainResource creates a new DomainResource
func NewDomainResource(item types.DomainStatus) *DomainResource {
	domainName := appaws.Str(item.DomainName)
	arn := appaws.Str(item.ARN)

	return &DomainResource{
		BaseResource: dao.BaseResource{
			ID:   domainName,
			Name: domainName,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: item,
		},
		Item: item,
	}
}

// DomainName returns the domain name
func (r *DomainResource) DomainName() string {
	return appaws.Str(r.Item.DomainName)
}

// DomainId returns the domain ID
func (r *DomainResource) DomainId() string {
	return appaws.Str(r.Item.DomainId)
}

// EngineVersion returns the OpenSearch/Elasticsearch version
func (r *DomainResource) EngineVersion() string {
	return appaws.Str(r.Item.EngineVersion)
}

// Endpoint returns the domain endpoint
func (r *DomainResource) Endpoint() string {
	return appaws.Str(r.Item.Endpoint)
}

// Endpoints returns the VPC endpoints
func (r *DomainResource) Endpoints() map[string]string {
	return r.Item.Endpoints
}

// Processing returns whether the domain is being processed
func (r *DomainResource) Processing() bool {
	if r.Item.Processing != nil {
		return *r.Item.Processing
	}
	return false
}

// Created returns whether the domain has been created
func (r *DomainResource) Created() bool {
	if r.Item.Created != nil {
		return *r.Item.Created
	}
	return false
}

// Deleted returns whether the domain has been deleted
func (r *DomainResource) Deleted() bool {
	if r.Item.Deleted != nil {
		return *r.Item.Deleted
	}
	return false
}

// Status returns the domain status as a string
func (r *DomainResource) Status() string {
	if r.Deleted() {
		return "DELETED"
	}
	if r.Processing() {
		return "PROCESSING"
	}
	if r.Created() {
		return "ACTIVE"
	}
	return "CREATING"
}

// InstanceType returns the instance type
func (r *DomainResource) InstanceType() string {
	if r.Item.ClusterConfig != nil {
		return string(r.Item.ClusterConfig.InstanceType)
	}
	return ""
}

// InstanceCount returns the number of instances
func (r *DomainResource) InstanceCount() int32 {
	if r.Item.ClusterConfig != nil && r.Item.ClusterConfig.InstanceCount != nil {
		return *r.Item.ClusterConfig.InstanceCount
	}
	return 0
}

// DedicatedMasterEnabled returns whether dedicated master is enabled
func (r *DomainResource) DedicatedMasterEnabled() bool {
	if r.Item.ClusterConfig != nil && r.Item.ClusterConfig.DedicatedMasterEnabled != nil {
		return *r.Item.ClusterConfig.DedicatedMasterEnabled
	}
	return false
}

// DedicatedMasterType returns the dedicated master type
func (r *DomainResource) DedicatedMasterType() string {
	if r.Item.ClusterConfig != nil {
		return string(r.Item.ClusterConfig.DedicatedMasterType)
	}
	return ""
}

// DedicatedMasterCount returns the number of dedicated master nodes
func (r *DomainResource) DedicatedMasterCount() int32 {
	if r.Item.ClusterConfig != nil && r.Item.ClusterConfig.DedicatedMasterCount != nil {
		return *r.Item.ClusterConfig.DedicatedMasterCount
	}
	return 0
}

// ZoneAwarenessEnabled returns whether zone awareness is enabled
func (r *DomainResource) ZoneAwarenessEnabled() bool {
	if r.Item.ClusterConfig != nil && r.Item.ClusterConfig.ZoneAwarenessEnabled != nil {
		return *r.Item.ClusterConfig.ZoneAwarenessEnabled
	}
	return false
}

// WarmEnabled returns whether warm storage is enabled
func (r *DomainResource) WarmEnabled() bool {
	if r.Item.ClusterConfig != nil && r.Item.ClusterConfig.WarmEnabled != nil {
		return *r.Item.ClusterConfig.WarmEnabled
	}
	return false
}

// WarmCount returns the number of warm nodes
func (r *DomainResource) WarmCount() int32 {
	if r.Item.ClusterConfig != nil && r.Item.ClusterConfig.WarmCount != nil {
		return *r.Item.ClusterConfig.WarmCount
	}
	return 0
}

// WarmType returns the warm node type
func (r *DomainResource) WarmType() string {
	if r.Item.ClusterConfig != nil {
		return string(r.Item.ClusterConfig.WarmType)
	}
	return ""
}

// EBSEnabled returns whether EBS is enabled
func (r *DomainResource) EBSEnabled() bool {
	if r.Item.EBSOptions != nil && r.Item.EBSOptions.EBSEnabled != nil {
		return *r.Item.EBSOptions.EBSEnabled
	}
	return false
}

// VolumeType returns the EBS volume type
func (r *DomainResource) VolumeType() string {
	if r.Item.EBSOptions != nil {
		return string(r.Item.EBSOptions.VolumeType)
	}
	return ""
}

// VolumeSize returns the EBS volume size in GB
func (r *DomainResource) VolumeSize() int32 {
	if r.Item.EBSOptions != nil && r.Item.EBSOptions.VolumeSize != nil {
		return *r.Item.EBSOptions.VolumeSize
	}
	return 0
}

// EncryptionAtRestEnabled returns whether encryption at rest is enabled
func (r *DomainResource) EncryptionAtRestEnabled() bool {
	if r.Item.EncryptionAtRestOptions != nil && r.Item.EncryptionAtRestOptions.Enabled != nil {
		return *r.Item.EncryptionAtRestOptions.Enabled
	}
	return false
}

// NodeToNodeEncryptionEnabled returns whether node-to-node encryption is enabled
func (r *DomainResource) NodeToNodeEncryptionEnabled() bool {
	if r.Item.NodeToNodeEncryptionOptions != nil && r.Item.NodeToNodeEncryptionOptions.Enabled != nil {
		return *r.Item.NodeToNodeEncryptionOptions.Enabled
	}
	return false
}

// EnforceHTTPS returns whether HTTPS is enforced
func (r *DomainResource) EnforceHTTPS() bool {
	if r.Item.DomainEndpointOptions != nil && r.Item.DomainEndpointOptions.EnforceHTTPS != nil {
		return *r.Item.DomainEndpointOptions.EnforceHTTPS
	}
	return false
}

// TLSSecurityPolicy returns the TLS security policy
func (r *DomainResource) TLSSecurityPolicy() string {
	if r.Item.DomainEndpointOptions != nil {
		return string(r.Item.DomainEndpointOptions.TLSSecurityPolicy)
	}
	return ""
}

// AdvancedSecurityEnabled returns whether fine-grained access control is enabled
func (r *DomainResource) AdvancedSecurityEnabled() bool {
	if r.Item.AdvancedSecurityOptions != nil && r.Item.AdvancedSecurityOptions.Enabled != nil {
		return *r.Item.AdvancedSecurityOptions.Enabled
	}
	return false
}

// VPCId returns the VPC ID if in VPC mode
func (r *DomainResource) VPCId() string {
	if r.Item.VPCOptions != nil {
		return appaws.Str(r.Item.VPCOptions.VPCId)
	}
	return ""
}

// SubnetIds returns the subnet IDs
func (r *DomainResource) SubnetIds() []string {
	if r.Item.VPCOptions != nil {
		return r.Item.VPCOptions.SubnetIds
	}
	return nil
}

// SecurityGroupIds returns the security group IDs
func (r *DomainResource) SecurityGroupIds() []string {
	if r.Item.VPCOptions != nil {
		return r.Item.VPCOptions.SecurityGroupIds
	}
	return nil
}

// AutoTuneState returns the auto-tune state
func (r *DomainResource) AutoTuneState() string {
	if r.Item.AutoTuneOptions != nil {
		return string(r.Item.AutoTuneOptions.State)
	}
	return ""
}
