package scripts

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ScriptDAO provides data access for GameLift scripts.
type ScriptDAO struct {
	dao.BaseDAO
	client *gamelift.Client
}

// NewScriptDAO creates a new ScriptDAO.
func NewScriptDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ScriptDAO{
		BaseDAO: dao.NewBaseDAO("gamelift", "scripts"),
		client:  gamelift.NewFromConfig(cfg),
	}, nil
}

// List returns all GameLift scripts.
func (d *ScriptDAO) List(ctx context.Context) ([]dao.Resource, error) {
	scripts, err := appaws.Paginate(ctx, func(token *string) ([]types.Script, *string, error) {
		output, err := d.client.ListScripts(ctx, &gamelift.ListScriptsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list gamelift scripts")
		}
		return output.Scripts, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(scripts))
	for i, script := range scripts {
		resources[i] = NewScriptResource(script)
	}
	return resources, nil
}

// Get returns a specific GameLift script by ID.
func (d *ScriptDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeScript(ctx, &gamelift.DescribeScriptInput{
		ScriptId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe gamelift script %s", id)
	}
	if output.Script == nil {
		return nil, fmt.Errorf("gamelift script %s not found", id)
	}
	return NewScriptResource(*output.Script), nil
}

// Delete deletes a GameLift script by ID.
func (d *ScriptDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteScript(ctx, &gamelift.DeleteScriptInput{
		ScriptId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete gamelift script %s", id)
	}
	return nil
}

// ScriptResource wraps a GameLift script.
type ScriptResource struct {
	dao.BaseResource
	Script types.Script
}

// NewScriptResource creates a new ScriptResource.
func NewScriptResource(script types.Script) *ScriptResource {
	return &ScriptResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(script.ScriptId),
			Name: appaws.Str(script.Name),
			ARN:  appaws.Str(script.ScriptArn),
			Data: script,
		},
		Script: script,
	}
}

// Version returns the script version.
func (r *ScriptResource) Version() string {
	return appaws.Str(r.Script.Version)
}

// SizeOnDisk returns the size in bytes.
func (r *ScriptResource) SizeOnDisk() int64 {
	return appaws.Int64(r.Script.SizeOnDisk)
}

// CreationTime returns when the script was created.
func (r *ScriptResource) CreationTime() *time.Time {
	return r.Script.CreationTime
}

// NodeJsVersion returns the Node.js version.
func (r *ScriptResource) NodeJsVersion() string {
	return appaws.Str(r.Script.NodeJsVersion)
}

// StorageLocationBucket returns the S3 bucket.
func (r *ScriptResource) StorageLocationBucket() string {
	if r.Script.StorageLocation != nil {
		return appaws.Str(r.Script.StorageLocation.Bucket)
	}
	return ""
}

// StorageLocationKey returns the S3 key.
func (r *ScriptResource) StorageLocationKey() string {
	if r.Script.StorageLocation != nil {
		return appaws.Str(r.Script.StorageLocation.Key)
	}
	return ""
}
