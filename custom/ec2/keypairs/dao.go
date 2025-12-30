package keypairs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// KeyPairDAO provides data access for EC2 Key Pairs
type KeyPairDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewKeyPairDAO creates a new KeyPairDAO
func NewKeyPairDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ec2/keypairs dao")
	}
	return &KeyPairDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "key-pairs"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *KeyPairDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe key pairs")
	}

	var resources []dao.Resource
	for _, kp := range output.KeyPairs {
		resources = append(resources, NewKeyPairResource(kp))
	}

	return resources, nil
}

func (d *KeyPairDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyPairIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe key pair %s", id)
	}

	if len(output.KeyPairs) == 0 {
		return nil, fmt.Errorf("key pair not found: %s", id)
	}

	return NewKeyPairResource(output.KeyPairs[0]), nil
}

func (d *KeyPairDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyPairId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete key pair %s", id)
	}

	return nil
}

// KeyPairResource wraps an EC2 Key Pair
type KeyPairResource struct {
	dao.BaseResource
	Item types.KeyPairInfo
}

// NewKeyPairResource creates a new KeyPairResource
func NewKeyPairResource(kp types.KeyPairInfo) *KeyPairResource {
	return &KeyPairResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(kp.KeyPairId),
			Name: appaws.Str(kp.KeyName),
			Tags: appaws.TagsToMap(kp.Tags),
			Data: kp,
		},
		Item: kp,
	}
}

func (r *KeyPairResource) KeyName() string {
	if r.Item.KeyName != nil {
		return *r.Item.KeyName
	}
	return ""
}

func (r *KeyPairResource) KeyType() string {
	return string(r.Item.KeyType)
}

func (r *KeyPairResource) Fingerprint() string {
	if r.Item.KeyFingerprint != nil {
		return *r.Item.KeyFingerprint
	}
	return ""
}

func (r *KeyPairResource) PublicKey() string {
	if r.Item.PublicKey != nil {
		return *r.Item.PublicKey
	}
	return ""
}
