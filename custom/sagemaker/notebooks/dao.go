package notebooks

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// NotebookDAO provides data access for SageMaker notebook instances.
type NotebookDAO struct {
	dao.BaseDAO
	client *sagemaker.Client
}

// NewNotebookDAO creates a new NotebookDAO.
func NewNotebookDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new sagemaker/notebooks dao: %w", err)
	}
	return &NotebookDAO{
		BaseDAO: dao.NewBaseDAO("sagemaker", "notebooks"),
		client:  sagemaker.NewFromConfig(cfg),
	}, nil
}

// List returns all SageMaker notebook instances.
func (d *NotebookDAO) List(ctx context.Context) ([]dao.Resource, error) {
	notebooks, err := appaws.Paginate(ctx, func(token *string) ([]types.NotebookInstanceSummary, *string, error) {
		output, err := d.client.ListNotebookInstances(ctx, &sagemaker.ListNotebookInstancesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list sagemaker notebooks: %w", err)
		}
		return output.NotebookInstances, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(notebooks))
	for i, notebook := range notebooks {
		resources[i] = NewNotebookResource(notebook)
	}
	return resources, nil
}

// Get returns a specific notebook instance.
func (d *NotebookDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{
		NotebookInstanceName: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("describe sagemaker notebook: %w", err)
	}
	// Convert to summary for consistent resource type
	summary := types.NotebookInstanceSummary{
		NotebookInstanceName:   output.NotebookInstanceName,
		NotebookInstanceArn:    output.NotebookInstanceArn,
		NotebookInstanceStatus: output.NotebookInstanceStatus,
		InstanceType:           output.InstanceType,
		CreationTime:           output.CreationTime,
		LastModifiedTime:       output.LastModifiedTime,
	}
	r := NewNotebookResource(summary)
	r.URL = appaws.Str(output.Url)
	r.RoleArn = appaws.Str(output.RoleArn)
	r.SubnetId = appaws.Str(output.SubnetId)
	r.SecurityGroups = output.SecurityGroups
	r.DirectInternetAccess = string(output.DirectInternetAccess)
	if output.VolumeSizeInGB != nil {
		r.VolumeSizeInGB = *output.VolumeSizeInGB
	}
	r.KmsKeyId = appaws.Str(output.KmsKeyId)
	r.LifecycleConfigName = appaws.Str(output.NotebookInstanceLifecycleConfigName)
	r.DefaultCodeRepository = appaws.Str(output.DefaultCodeRepository)
	r.AdditionalCodeRepositories = output.AdditionalCodeRepositories
	r.FailureReason = appaws.Str(output.FailureReason)
	r.PlatformIdentifier = appaws.Str(output.PlatformIdentifier)
	r.RootAccess = string(output.RootAccess)
	r.NetworkInterfaceId = appaws.Str(output.NetworkInterfaceId)
	return r, nil
}

// Delete deletes a notebook instance.
func (d *NotebookDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteNotebookInstance(ctx, &sagemaker.DeleteNotebookInstanceInput{
		NotebookInstanceName: &id,
	})
	if err != nil {
		return fmt.Errorf("delete sagemaker notebook: %w", err)
	}
	return nil
}

// NotebookResource wraps a SageMaker notebook instance.
type NotebookResource struct {
	dao.BaseResource
	Notebook                   types.NotebookInstanceSummary
	URL                        string
	RoleArn                    string
	SubnetId                   string
	SecurityGroups             []string
	DirectInternetAccess       string
	VolumeSizeInGB             int32
	KmsKeyId                   string
	LifecycleConfigName        string
	DefaultCodeRepository      string
	AdditionalCodeRepositories []string
	FailureReason              string
	PlatformIdentifier         string
	RootAccess                 string
	NetworkInterfaceId         string
}

// NewNotebookResource creates a new NotebookResource.
func NewNotebookResource(notebook types.NotebookInstanceSummary) *NotebookResource {
	return &NotebookResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(notebook.NotebookInstanceName),
			ARN: appaws.Str(notebook.NotebookInstanceArn),
		},
		Notebook: notebook,
	}
}

// Status returns the notebook status.
func (r *NotebookResource) Status() string {
	return string(r.Notebook.NotebookInstanceStatus)
}

// InstanceType returns the instance type.
func (r *NotebookResource) InstanceType() string {
	return string(r.Notebook.InstanceType)
}

// CreatedAt returns when the notebook was created.
func (r *NotebookResource) CreatedAt() *time.Time {
	return r.Notebook.CreationTime
}

// LastModifiedAt returns when the notebook was last modified.
func (r *NotebookResource) LastModifiedAt() *time.Time {
	return r.Notebook.LastModifiedTime
}

// GetURL returns the notebook URL.
func (r *NotebookResource) GetURL() string {
	return r.URL
}

// GetRoleArn returns the IAM role ARN.
func (r *NotebookResource) GetRoleArn() string {
	return r.RoleArn
}

// GetSubnetId returns the subnet ID.
func (r *NotebookResource) GetSubnetId() string {
	return r.SubnetId
}

// GetSecurityGroups returns the security groups.
func (r *NotebookResource) GetSecurityGroups() []string {
	return r.SecurityGroups
}

// GetDirectInternetAccess returns the direct internet access setting.
func (r *NotebookResource) GetDirectInternetAccess() string {
	return r.DirectInternetAccess
}

// GetVolumeSizeInGB returns the volume size.
func (r *NotebookResource) GetVolumeSizeInGB() int32 {
	return r.VolumeSizeInGB
}

// GetKmsKeyId returns the KMS key ID.
func (r *NotebookResource) GetKmsKeyId() string {
	return r.KmsKeyId
}

// GetLifecycleConfigName returns the lifecycle config name.
func (r *NotebookResource) GetLifecycleConfigName() string {
	return r.LifecycleConfigName
}

// GetDefaultCodeRepository returns the default code repository.
func (r *NotebookResource) GetDefaultCodeRepository() string {
	return r.DefaultCodeRepository
}

// GetAdditionalCodeRepositories returns additional code repositories.
func (r *NotebookResource) GetAdditionalCodeRepositories() []string {
	return r.AdditionalCodeRepositories
}

// GetFailureReason returns the failure reason.
func (r *NotebookResource) GetFailureReason() string {
	return r.FailureReason
}

// GetPlatformIdentifier returns the platform identifier.
func (r *NotebookResource) GetPlatformIdentifier() string {
	return r.PlatformIdentifier
}

// GetRootAccess returns the root access setting.
func (r *NotebookResource) GetRootAccess() string {
	return r.RootAccess
}

// GetNetworkInterfaceId returns the network interface ID.
func (r *NotebookResource) GetNetworkInterfaceId() string {
	return r.NetworkInterfaceId
}
