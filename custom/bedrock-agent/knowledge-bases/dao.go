package knowledgebases

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// KnowledgeBaseDAO provides data access for Bedrock Knowledge Bases
type KnowledgeBaseDAO struct {
	dao.BaseDAO
	client *bedrockagent.Client
}

// NewKnowledgeBaseDAO creates a new KnowledgeBaseDAO
func NewKnowledgeBaseDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agent/knowledge-bases dao")
	}
	return &KnowledgeBaseDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agent", "knowledge-bases"),
		client:  bedrockagent.NewFromConfig(cfg),
	}, nil
}

// Client returns the bedrockagent client for shared use
func (d *KnowledgeBaseDAO) Client() *bedrockagent.Client {
	return d.client
}

func (d *KnowledgeBaseDAO) List(ctx context.Context) ([]dao.Resource, error) {
	knowledgeBases, err := appaws.Paginate(ctx, func(token *string) ([]types.KnowledgeBaseSummary, *string, error) {
		output, err := d.client.ListKnowledgeBases(ctx, &bedrockagent.ListKnowledgeBasesInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list knowledge bases")
		}
		return output.KnowledgeBaseSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(knowledgeBases))
	for i, kb := range knowledgeBases {
		resources[i] = NewKnowledgeBaseResource(kb)
	}

	return resources, nil
}

func (d *KnowledgeBaseDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetKnowledgeBase(ctx, &bedrockagent.GetKnowledgeBaseInput{
		KnowledgeBaseId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get knowledge base %s", id)
	}

	return NewKnowledgeBaseResourceFromDetail(output.KnowledgeBase), nil
}

func (d *KnowledgeBaseDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteKnowledgeBase(ctx, &bedrockagent.DeleteKnowledgeBaseInput{
		KnowledgeBaseId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete knowledge base %s", id)
	}
	return nil
}

// KnowledgeBaseResource wraps a Bedrock Knowledge Base
type KnowledgeBaseResource struct {
	dao.BaseResource
	Item       types.KnowledgeBaseSummary
	DetailItem *types.KnowledgeBase
	IsFromList bool
}

// NewKnowledgeBaseResource creates a new KnowledgeBaseResource from list output
func NewKnowledgeBaseResource(kb types.KnowledgeBaseSummary) *KnowledgeBaseResource {
	id := appaws.Str(kb.KnowledgeBaseId)
	name := appaws.Str(kb.Name)

	return &KnowledgeBaseResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  "", // ARN not available in summary
			Data: kb,
		},
		Item:       kb,
		IsFromList: true,
	}
}

// NewKnowledgeBaseResourceFromDetail creates a KnowledgeBaseResource from detail output
func NewKnowledgeBaseResourceFromDetail(kb *types.KnowledgeBase) *KnowledgeBaseResource {
	id := appaws.Str(kb.KnowledgeBaseId)
	name := appaws.Str(kb.Name)
	arn := appaws.Str(kb.KnowledgeBaseArn)

	return &KnowledgeBaseResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: kb,
		},
		DetailItem: kb,
		IsFromList: false,
	}
}

// Status returns the knowledge base status
func (r *KnowledgeBaseResource) Status() string {
	if r.IsFromList {
		return string(r.Item.Status)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return ""
}

// Description returns the knowledge base description
func (r *KnowledgeBaseResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *KnowledgeBaseResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *KnowledgeBaseResource) CreatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// RoleArn returns the IAM role ARN
func (r *KnowledgeBaseResource) RoleArn() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.RoleArn)
	}
	return ""
}

// EmbeddingModelArn returns the embedding model ARN
func (r *KnowledgeBaseResource) EmbeddingModelArn() string {
	if r.DetailItem != nil && r.DetailItem.KnowledgeBaseConfiguration != nil {
		if vec := r.DetailItem.KnowledgeBaseConfiguration.VectorKnowledgeBaseConfiguration; vec != nil {
			return appaws.Str(vec.EmbeddingModelArn)
		}
	}
	return ""
}

// StorageType returns the storage configuration type
func (r *KnowledgeBaseResource) StorageType() string {
	if r.DetailItem != nil && r.DetailItem.StorageConfiguration != nil {
		return string(r.DetailItem.StorageConfiguration.Type)
	}
	return ""
}

// FailureReasons returns any failure reasons
func (r *KnowledgeBaseResource) FailureReasons() []string {
	if r.DetailItem != nil {
		return r.DetailItem.FailureReasons
	}
	return nil
}
