package functions

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FunctionDAO provides data access for Lambda functions
type FunctionDAO struct {
	dao.BaseDAO
	client *lambda.Client
}

// NewFunctionDAO creates a new FunctionDAO
func NewFunctionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &FunctionDAO{
		BaseDAO: dao.NewBaseDAO("lambda", "functions"),
		client:  lambda.NewFromConfig(cfg),
	}, nil
}

// List returns functions (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *FunctionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of Lambda functions.
// Implements dao.PaginatedDAO interface.
func (d *FunctionDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxItems := int32(pageSize)
	if maxItems > 50 {
		maxItems = 50 // AWS API max
	}

	input := &lambda.ListFunctionsInput{
		MaxItems: &maxItems,
	}
	if pageToken != "" {
		input.Marker = &pageToken
	}

	output, err := d.client.ListFunctions(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list functions")
	}

	resources := make([]dao.Resource, len(output.Functions))
	for i, fn := range output.Functions {
		resources[i] = NewFunctionResource(fn)
	}

	nextToken := ""
	if output.NextMarker != nil {
		nextToken = *output.NextMarker
	}

	return resources, nextToken, nil
}

func (d *FunctionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &lambda.GetFunctionInput{
		FunctionName: &id,
	}

	output, err := d.client.GetFunction(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get function %s", id)
	}

	if output.Configuration == nil {
		return nil, fmt.Errorf("function not found: %s", id)
	}

	res := NewFunctionResourceFromConfig(*output.Configuration)

	// Fetch concurrency configuration
	if concurrency, err := d.client.GetFunctionConcurrency(ctx, &lambda.GetFunctionConcurrencyInput{
		FunctionName: &id,
	}); err == nil && concurrency.ReservedConcurrentExecutions != nil {
		res.ReservedConcurrency = concurrency.ReservedConcurrentExecutions
	}

	// Fetch function URL (if exists)
	if urlConfig, err := d.client.GetFunctionUrlConfig(ctx, &lambda.GetFunctionUrlConfigInput{
		FunctionName: &id,
	}); err == nil && urlConfig.FunctionUrl != nil {
		res.FunctionURL = *urlConfig.FunctionUrl
	}

	return res, nil
}

func (d *FunctionDAO) Delete(ctx context.Context, id string) error {
	input := &lambda.DeleteFunctionInput{
		FunctionName: &id,
	}

	_, err := d.client.DeleteFunction(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "function %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete function %s", id)
	}

	return nil
}

// FunctionResource wraps a Lambda function
type FunctionResource struct {
	dao.BaseResource
	Item                   types.FunctionConfiguration
	ReservedConcurrency    *int32
	ProvisionedConcurrency *int32
	FunctionURL            string
}

// NewFunctionResource creates a new FunctionResource from ListFunctions output
func NewFunctionResource(fn types.FunctionConfiguration) *FunctionResource {
	return NewFunctionResourceFromConfig(fn)
}

// NewFunctionResourceFromConfig creates a new FunctionResource from FunctionConfiguration
func NewFunctionResourceFromConfig(fn types.FunctionConfiguration) *FunctionResource {
	return &FunctionResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(fn.FunctionName),
			Name: appaws.Str(fn.FunctionName),
			ARN:  appaws.Str(fn.FunctionArn),
			Data: fn,
		},
		Item: fn,
	}
}

// Runtime returns the runtime
func (r *FunctionResource) Runtime() string {
	return string(r.Item.Runtime)
}

// State returns the function state
func (r *FunctionResource) State() string {
	return string(r.Item.State)
}

// MemorySize returns the memory size in MB
func (r *FunctionResource) MemorySize() int32 {
	if r.Item.MemorySize != nil {
		return *r.Item.MemorySize
	}
	return 0
}

// Timeout returns the timeout in seconds
func (r *FunctionResource) Timeout() int32 {
	if r.Item.Timeout != nil {
		return *r.Item.Timeout
	}
	return 0
}

// CodeSize returns the code size in bytes
func (r *FunctionResource) CodeSize() int64 {
	return r.Item.CodeSize
}

// Handler returns the handler
func (r *FunctionResource) Handler() string {
	return appaws.Str(r.Item.Handler)
}

// Description returns the description
func (r *FunctionResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// LastModified returns the last modified time
func (r *FunctionResource) LastModified() string {
	return appaws.Str(r.Item.LastModified)
}

// Role returns the execution role ARN
func (r *FunctionResource) Role() string {
	return appaws.Str(r.Item.Role)
}

// PackageType returns the package type
func (r *FunctionResource) PackageType() string {
	return string(r.Item.PackageType)
}

// Architectures returns the architectures
func (r *FunctionResource) Architectures() []types.Architecture {
	return r.Item.Architectures
}

// DeadLetterConfig returns the dead letter queue config
func (r *FunctionResource) DeadLetterConfig() *types.DeadLetterConfig {
	return r.Item.DeadLetterConfig
}

// EphemeralStorageSize returns the /tmp directory size in MB
func (r *FunctionResource) EphemeralStorageSize() int32 {
	if r.Item.EphemeralStorage != nil && r.Item.EphemeralStorage.Size != nil {
		return *r.Item.EphemeralStorage.Size
	}
	return 512 // default
}

// TracingConfig returns the X-Ray tracing mode
func (r *FunctionResource) TracingConfig() string {
	if r.Item.TracingConfig != nil {
		return string(r.Item.TracingConfig.Mode)
	}
	return ""
}

// Layers returns the function layers
func (r *FunctionResource) Layers() []types.Layer {
	return r.Item.Layers
}

// StateReason returns the reason for the function state
func (r *FunctionResource) StateReason() string {
	return appaws.Str(r.Item.StateReason)
}

// StateReasonCode returns the state reason code
func (r *FunctionResource) StateReasonCode() string {
	return string(r.Item.StateReasonCode)
}

// SnapStart returns the SnapStart configuration
func (r *FunctionResource) SnapStart() *types.SnapStartResponse {
	return r.Item.SnapStart
}

// KMSKeyArn returns the KMS key ARN for environment encryption
func (r *FunctionResource) KMSKeyArn() string {
	return appaws.Str(r.Item.KMSKeyArn)
}

// CodeSha256 returns the SHA256 hash of the deployment package
func (r *FunctionResource) CodeSha256() string {
	return appaws.Str(r.Item.CodeSha256)
}

// Version returns the function version
func (r *FunctionResource) Version() string {
	return appaws.Str(r.Item.Version)
}
