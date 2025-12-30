package repositories

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// RepositoryDAO provides data access for ECR repositories
type RepositoryDAO struct {
	dao.BaseDAO
	client *ecr.Client
}

// NewRepositoryDAO creates a new RepositoryDAO
func NewRepositoryDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ecr/repositories dao: %w", err)
	}
	return &RepositoryDAO{
		BaseDAO: dao.NewBaseDAO("ecr", "repositories"),
		client:  ecr.NewFromConfig(cfg),
	}, nil
}

func (d *RepositoryDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ecr.DescribeRepositoriesInput{}
	paginator := ecr.NewDescribeRepositoriesPaginator(d.client, input)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe repositories: %w", err)
		}

		for _, repo := range output.Repositories {
			resources = append(resources, NewRepositoryResource(repo))
		}
	}

	return resources, nil
}

func (d *RepositoryDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{id},
	}

	output, err := d.client.DescribeRepositories(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe repository %s: %w", id, err)
	}

	if len(output.Repositories) == 0 {
		return nil, fmt.Errorf("repository not found: %s", id)
	}

	return NewRepositoryResource(output.Repositories[0]), nil
}

func (d *RepositoryDAO) Delete(ctx context.Context, id string) error {
	input := &ecr.DeleteRepositoryInput{
		RepositoryName: &id,
		Force:          true,
	}

	_, err := d.client.DeleteRepository(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("repository %s is in use", id)
		}
		return fmt.Errorf("delete repository %s: %w", id, err)
	}

	return nil
}

// RepositoryResource wraps an ECR repository
type RepositoryResource struct {
	dao.BaseResource
	Item types.Repository
}

// NewRepositoryResource creates a new RepositoryResource
func NewRepositoryResource(repo types.Repository) *RepositoryResource {
	name := appaws.Str(repo.RepositoryName)

	return &RepositoryResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(repo.RepositoryArn),
			Tags: nil, // ECR repos don't have tags in the describe output
			Data: repo,
		},
		Item: repo,
	}
}

// URI returns the repository URI
func (r *RepositoryResource) URI() string {
	if r.Item.RepositoryUri != nil {
		return *r.Item.RepositoryUri
	}
	return ""
}

// ARN returns the repository ARN
func (r *RepositoryResource) ARN() string {
	if r.Item.RepositoryArn != nil {
		return *r.Item.RepositoryArn
	}
	return ""
}

// ScanOnPush returns whether image scanning on push is enabled
func (r *RepositoryResource) ScanOnPush() bool {
	if r.Item.ImageScanningConfiguration != nil {
		return r.Item.ImageScanningConfiguration.ScanOnPush
	}
	return false
}

// ImageTagMutability returns the image tag mutability setting
func (r *RepositoryResource) ImageTagMutability() string {
	return string(r.Item.ImageTagMutability)
}

// EncryptionType returns the encryption type
func (r *RepositoryResource) EncryptionType() string {
	if r.Item.EncryptionConfiguration != nil {
		return string(r.Item.EncryptionConfiguration.EncryptionType)
	}
	return ""
}
