package tgwattachments

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

// TGWAttachmentDAO provides data access for Transit Gateway attachments.
type TGWAttachmentDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewTGWAttachmentDAO creates a new TGWAttachmentDAO.
func NewTGWAttachmentDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new vpc/tgw-attachments dao")
	}
	return &TGWAttachmentDAO{
		BaseDAO: dao.NewBaseDAO("vpc", "tgw-attachments"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

// List returns all Transit Gateway attachments, optionally filtered by TGW ID.
func (d *TGWAttachmentDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeTransitGatewayAttachmentsInput{}

	// Filter by Transit Gateway ID if provided
	if tgwID := dao.GetFilterFromContext(ctx, "TransitGatewayId"); tgwID != "" {
		input.Filters = []types.Filter{
			{
				Name:   appaws.StringPtr("transit-gateway-id"),
				Values: []string{tgwID},
			},
		}
	}

	attachments, err := appaws.Paginate(ctx, func(token *string) ([]types.TransitGatewayAttachment, *string, error) {
		input.NextToken = token
		output, err := d.client.DescribeTransitGatewayAttachments(ctx, input)
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe transit gateway attachments")
		}
		return output.TransitGatewayAttachments, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(attachments))
	for i, att := range attachments {
		resources[i] = NewTGWAttachmentResource(att)
	}
	return resources, nil
}

// Get returns a specific Transit Gateway attachment by ID.
func (d *TGWAttachmentDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeTransitGatewayAttachments(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{
		TransitGatewayAttachmentIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe transit gateway attachment %s", id)
	}
	if len(output.TransitGatewayAttachments) == 0 {
		return nil, fmt.Errorf("transit gateway attachment not found: %s", id)
	}
	return NewTGWAttachmentResource(output.TransitGatewayAttachments[0]), nil
}

// Delete deletes a Transit Gateway attachment by ID.
func (d *TGWAttachmentDAO) Delete(ctx context.Context, id string) error {
	// First get the attachment to determine its type
	att, err := d.Get(ctx, id)
	if err != nil {
		return err
	}
	attRes := att.(*TGWAttachmentResource)

	switch attRes.ResourceType() {
	case "vpc":
		_, err = d.client.DeleteTransitGatewayVpcAttachment(ctx, &ec2.DeleteTransitGatewayVpcAttachmentInput{
			TransitGatewayAttachmentId: &id,
		})
	case "peering":
		_, err = d.client.DeleteTransitGatewayPeeringAttachment(ctx, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
			TransitGatewayAttachmentId: &id,
		})
	default:
		return fmt.Errorf("cannot delete attachment of type %s", attRes.ResourceType())
	}

	if err != nil {
		return apperrors.Wrapf(err, "delete transit gateway attachment %s", id)
	}
	return nil
}

// TGWAttachmentResource wraps a Transit Gateway attachment.
type TGWAttachmentResource struct {
	dao.BaseResource
	Item types.TransitGatewayAttachment
}

// NewTGWAttachmentResource creates a new TGWAttachmentResource.
func NewTGWAttachmentResource(att types.TransitGatewayAttachment) *TGWAttachmentResource {
	return &TGWAttachmentResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(att.TransitGatewayAttachmentId),
			ARN: "",
		},
		Item: att,
	}
}

// TransitGatewayId returns the TGW ID.
func (r *TGWAttachmentResource) TransitGatewayId() string {
	return appaws.Str(r.Item.TransitGatewayId)
}

// ResourceType returns the attachment resource type.
func (r *TGWAttachmentResource) ResourceType() string {
	return string(r.Item.ResourceType)
}

// ResourceId returns the attached resource ID.
func (r *TGWAttachmentResource) ResourceId() string {
	return appaws.Str(r.Item.ResourceId)
}

// ResourceOwnerId returns the resource owner account ID.
func (r *TGWAttachmentResource) ResourceOwnerId() string {
	return appaws.Str(r.Item.ResourceOwnerId)
}

// State returns the attachment state.
func (r *TGWAttachmentResource) State() string {
	return string(r.Item.State)
}

// Association returns the association info.
func (r *TGWAttachmentResource) Association() string {
	if r.Item.Association != nil {
		return appaws.Str(r.Item.Association.TransitGatewayRouteTableId)
	}
	return ""
}

// CreationTime returns when the attachment was created.
func (r *TGWAttachmentResource) CreationTime() *time.Time {
	return r.Item.CreationTime
}

// Name returns the Name tag value.
func (r *TGWAttachmentResource) Name() string {
	for _, tag := range r.Item.Tags {
		if appaws.Str(tag.Key) == "Name" {
			return appaws.Str(tag.Value)
		}
	}
	return ""
}

// TransitGatewayOwnerId returns the TGW owner account ID.
func (r *TGWAttachmentResource) TransitGatewayOwnerId() string {
	return appaws.Str(r.Item.TransitGatewayOwnerId)
}

// AssociationState returns the association state.
func (r *TGWAttachmentResource) AssociationState() string {
	if r.Item.Association != nil {
		return string(r.Item.Association.State)
	}
	return ""
}

// Tags returns all tags.
func (r *TGWAttachmentResource) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range r.Item.Tags {
		tags[appaws.Str(tag.Key)] = appaws.Str(tag.Value)
	}
	return tags
}
