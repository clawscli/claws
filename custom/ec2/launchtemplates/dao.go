package launchtemplates

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// LaunchTemplateDAO provides data access for EC2 Launch Templates
type LaunchTemplateDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewLaunchTemplateDAO creates a new LaunchTemplateDAO
func NewLaunchTemplateDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ec2/launchtemplates dao")
	}
	return &LaunchTemplateDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "launch-templates"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

// List returns all Launch Templates
func (d *LaunchTemplateDAO) List(ctx context.Context) ([]dao.Resource, error) {
	templates, err := appaws.Paginate(ctx, func(token *string) ([]types.LaunchTemplate, *string, error) {
		output, err := d.client.DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list launch templates")
		}
		return output.LaunchTemplates, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(templates))
	for i, lt := range templates {
		resources[i] = NewLaunchTemplateResource(lt)
	}

	return resources, nil
}

// Get returns a specific Launch Template
func (d *LaunchTemplateDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get launch template %s", id)
	}

	if len(output.LaunchTemplates) == 0 {
		return nil, fmt.Errorf("launch template not found: %s", id)
	}

	return NewLaunchTemplateResource(output.LaunchTemplates[0]), nil
}

// Delete deletes a Launch Template
func (d *LaunchTemplateDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteLaunchTemplate(ctx, &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete launch template %s", id)
	}
	return nil
}

// LaunchTemplateResource wraps an EC2 Launch Template
type LaunchTemplateResource struct {
	dao.BaseResource
	Item types.LaunchTemplate
}

// NewLaunchTemplateResource creates a new LaunchTemplateResource
func NewLaunchTemplateResource(lt types.LaunchTemplate) *LaunchTemplateResource {
	return &LaunchTemplateResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(lt.LaunchTemplateId),
			Name: appaws.Str(lt.LaunchTemplateName),
			Tags: appaws.TagsToMap(lt.Tags),
			Data: lt,
		},
		Item: lt,
	}
}

// LaunchTemplateId returns the launch template ID
func (r *LaunchTemplateResource) LaunchTemplateId() string {
	if r.Item.LaunchTemplateId != nil {
		return *r.Item.LaunchTemplateId
	}
	return ""
}

// LaunchTemplateName returns the launch template name
func (r *LaunchTemplateResource) LaunchTemplateName() string {
	if r.Item.LaunchTemplateName != nil {
		return *r.Item.LaunchTemplateName
	}
	return ""
}

// DefaultVersionNumber returns the default version number
func (r *LaunchTemplateResource) DefaultVersionNumber() int64 {
	if r.Item.DefaultVersionNumber != nil {
		return *r.Item.DefaultVersionNumber
	}
	return 0
}

// LatestVersionNumber returns the latest version number
func (r *LaunchTemplateResource) LatestVersionNumber() int64 {
	if r.Item.LatestVersionNumber != nil {
		return *r.Item.LatestVersionNumber
	}
	return 0
}

// CreatedBy returns who created the template
func (r *LaunchTemplateResource) CreatedBy() string {
	if r.Item.CreatedBy != nil {
		return *r.Item.CreatedBy
	}
	return ""
}

// CreateTime returns the creation time
func (r *LaunchTemplateResource) CreateTime() time.Time {
	if r.Item.CreateTime != nil {
		return *r.Item.CreateTime
	}
	return time.Time{}
}
