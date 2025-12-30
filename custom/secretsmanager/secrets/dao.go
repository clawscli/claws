package secrets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// SecretDAO provides data access for Secrets Manager secrets
type SecretDAO struct {
	dao.BaseDAO
	client *secretsmanager.Client
}

// NewSecretDAO creates a new SecretDAO
func NewSecretDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new secretsmanager/secrets dao: %w", err)
	}
	return &SecretDAO{
		BaseDAO: dao.NewBaseDAO("secretsmanager", "secrets"),
		client:  secretsmanager.NewFromConfig(cfg),
	}, nil
}

func (d *SecretDAO) List(ctx context.Context) ([]dao.Resource, error) {
	secrets, err := appaws.Paginate(ctx, func(token *string) ([]types.SecretListEntry, *string, error) {
		output, err := d.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list secrets: %w", err)
		}
		return output.SecretList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(secrets))
	for i, secret := range secrets {
		resources[i] = NewSecretResource(secret)
	}
	return resources, nil
}

func (d *SecretDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &secretsmanager.DescribeSecretInput{
		SecretId: &id,
	}

	output, err := d.client.DescribeSecret(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe secret %s: %w", id, err)
	}

	// Convert DescribeSecretOutput to SecretListEntry for consistency
	secret := types.SecretListEntry{
		ARN:                    output.ARN,
		Name:                   output.Name,
		Description:            output.Description,
		LastChangedDate:        output.LastChangedDate,
		LastAccessedDate:       output.LastAccessedDate,
		Tags:                   output.Tags,
		SecretVersionsToStages: output.VersionIdsToStages,
		CreatedDate:            output.CreatedDate,
	}

	res := NewSecretResource(secret)

	// Capture additional fields from DescribeSecretOutput
	if output.KmsKeyId != nil {
		res.KmsKeyId = *output.KmsKeyId
	}
	if output.RotationEnabled != nil {
		res.RotationEnabled = *output.RotationEnabled
	}
	if output.RotationLambdaARN != nil {
		res.RotationLambdaARN = *output.RotationLambdaARN
	}
	res.RotationRules = output.RotationRules
	if output.DeletedDate != nil {
		delDate := output.DeletedDate.Format("2006-01-02 15:04:05")
		res.DeletedDate = &delDate
	}
	if output.PrimaryRegion != nil {
		res.PrimaryRegion = *output.PrimaryRegion
	}

	return res, nil
}

func (d *SecretDAO) Delete(ctx context.Context, id string) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId:                   &id,
		ForceDeleteWithoutRecovery: appaws.BoolPtr(false), // Safe delete with recovery window
	}

	_, err := d.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("delete secret %s: %w", id, err)
	}

	return nil
}

// SecretResource wraps a Secrets Manager secret
type SecretResource struct {
	dao.BaseResource
	Item              types.SecretListEntry
	KmsKeyId          string
	RotationEnabled   bool
	RotationLambdaARN string
	RotationRules     *types.RotationRulesType
	DeletedDate       *string
	PrimaryRegion     string
}

// NewSecretResource creates a new SecretResource
func NewSecretResource(secret types.SecretListEntry) *SecretResource {
	name := appaws.Str(secret.Name)
	arn := appaws.Str(secret.ARN)

	// Convert tags
	tags := make(map[string]string)
	for _, tag := range secret.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return &SecretResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: tags,
			Data: secret,
		},
		Item: secret,
	}
}

// Description returns the secret description
func (r *SecretResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// CreatedDate returns the creation date as string
func (r *SecretResource) CreatedDate() string {
	if r.Item.CreatedDate != nil {
		return r.Item.CreatedDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastChangedDate returns the last changed date as string
func (r *SecretResource) LastChangedDate() string {
	if r.Item.LastChangedDate != nil {
		return r.Item.LastChangedDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastAccessedDate returns the last accessed date as string
func (r *SecretResource) LastAccessedDate() string {
	if r.Item.LastAccessedDate != nil {
		return r.Item.LastAccessedDate.Format("2006-01-02")
	}
	return ""
}

// VersionCount returns the number of versions
func (r *SecretResource) VersionCount() int {
	return len(r.Item.SecretVersionsToStages)
}

// CurrentVersionId returns the current version ID
func (r *SecretResource) CurrentVersionId() string {
	for versionId, stages := range r.Item.SecretVersionsToStages {
		for _, stage := range stages {
			if stage == "AWSCURRENT" {
				return versionId
			}
		}
	}
	return ""
}
