package keys

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
)

// KeyDAO provides data access for KMS keys
type KeyDAO struct {
	dao.BaseDAO
	client *kms.Client
}

// NewKeyDAO creates a new KeyDAO
func NewKeyDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new kms/keys dao: %w", err)
	}
	return &KeyDAO{
		BaseDAO: dao.NewBaseDAO("kms", "keys"),
		client:  kms.NewFromConfig(cfg),
	}, nil
}

func (d *KeyDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource

	keyIter := appaws.PaginateIter(ctx, func(token *string) ([]types.KeyListEntry, *string, error) {
		output, err := d.client.ListKeys(ctx, &kms.ListKeysInput{
			Marker: token,
			Limit:  appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list keys: %w", err)
		}
		// KMS uses Truncated flag instead of checking NextMarker
		var nextToken *string
		if output.Truncated {
			nextToken = output.NextMarker
		}
		return output.Keys, nextToken, nil
	})

	for key, err := range keyIter {
		if err != nil {
			return nil, err
		}
		// Get key details
		describeOutput, err := d.client.DescribeKey(ctx, &kms.DescribeKeyInput{
			KeyId: key.KeyId,
		})
		if err != nil {
			log.Warn("failed to describe KMS key", "keyId", appaws.Str(key.KeyId), "error", err)
			continue
		}
		resources = append(resources, NewKeyResource(describeOutput.KeyMetadata))
	}

	return resources, nil
}

func (d *KeyDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &kms.DescribeKeyInput{
		KeyId: &id,
	}

	output, err := d.client.DescribeKey(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe key %s: %w", id, err)
	}

	return NewKeyResource(output.KeyMetadata), nil
}

func (d *KeyDAO) Delete(ctx context.Context, id string) error {
	// Schedule key deletion (minimum 7 days waiting period)
	pendingDays := int32(7)
	input := &kms.ScheduleKeyDeletionInput{
		KeyId:               &id,
		PendingWindowInDays: &pendingDays,
	}

	_, err := d.client.ScheduleKeyDeletion(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("key %s is in use", id)
		}
		return fmt.Errorf("schedule key deletion %s: %w", id, err)
	}

	return nil
}

// KeyResource wraps a KMS key
type KeyResource struct {
	dao.BaseResource
	Item *types.KeyMetadata
}

// NewKeyResource creates a new KeyResource
func NewKeyResource(key *types.KeyMetadata) *KeyResource {
	keyId := appaws.Str(key.KeyId)
	arn := appaws.Str(key.Arn)

	// Build display name (alias or key ID)
	name := keyId
	if key.Description != nil && *key.Description != "" {
		// Use first 30 chars of description if available
		desc := *key.Description
		if len(desc) > 30 {
			desc = desc[:30]
		}
		name = desc
	}

	return &KeyResource{
		BaseResource: dao.BaseResource{
			ID:   keyId,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: key,
		},
		Item: key,
	}
}

// KeyId returns the key ID
func (r *KeyResource) KeyId() string {
	return appaws.Str(r.Item.KeyId)
}

// Description returns the key description
func (r *KeyResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// KeyState returns the key state
func (r *KeyResource) KeyState() string {
	return string(r.Item.KeyState)
}

// KeySpec returns the key spec
func (r *KeyResource) KeySpec() string {
	return string(r.Item.KeySpec)
}

// KeyUsage returns the key usage
func (r *KeyResource) KeyUsage() string {
	return string(r.Item.KeyUsage)
}

// KeyManager returns the key manager (AWS or CUSTOMER)
func (r *KeyResource) KeyManager() string {
	return string(r.Item.KeyManager)
}

// Origin returns the key origin
func (r *KeyResource) Origin() string {
	return string(r.Item.Origin)
}

// CreationDate returns the creation date as string
func (r *KeyResource) CreationDate() string {
	if r.Item.CreationDate != nil {
		return r.Item.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// IsEnabled returns whether the key is enabled
func (r *KeyResource) IsEnabled() bool {
	return r.Item.Enabled
}

// IsMultiRegion returns whether the key is multi-region
func (r *KeyResource) IsMultiRegion() bool {
	return r.Item.MultiRegion != nil && *r.Item.MultiRegion
}

// DeletionDate returns the scheduled deletion date
func (r *KeyResource) DeletionDate() string {
	if r.Item.DeletionDate != nil {
		return r.Item.DeletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ValidTo returns the key material expiration date
func (r *KeyResource) ValidTo() string {
	if r.Item.ValidTo != nil {
		return r.Item.ValidTo.Format("2006-01-02 15:04:05")
	}
	return ""
}

// AWSAccountId returns the AWS account ID
func (r *KeyResource) AWSAccountId() string {
	return appaws.Str(r.Item.AWSAccountId)
}

// EncryptionAlgorithms returns the supported encryption algorithms
func (r *KeyResource) EncryptionAlgorithms() []string {
	algs := make([]string, len(r.Item.EncryptionAlgorithms))
	for i, alg := range r.Item.EncryptionAlgorithms {
		algs[i] = string(alg)
	}
	return algs
}

// SigningAlgorithms returns the supported signing algorithms
func (r *KeyResource) SigningAlgorithms() []string {
	algs := make([]string, len(r.Item.SigningAlgorithms))
	for i, alg := range r.Item.SigningAlgorithms {
		algs[i] = string(alg)
	}
	return algs
}

// MacAlgorithms returns the supported MAC algorithms
func (r *KeyResource) MacAlgorithms() []string {
	algs := make([]string, len(r.Item.MacAlgorithms))
	for i, alg := range r.Item.MacAlgorithms {
		algs[i] = string(alg)
	}
	return algs
}

// ExpirationModel returns the expiration model
func (r *KeyResource) ExpirationModel() string {
	return string(r.Item.ExpirationModel)
}
