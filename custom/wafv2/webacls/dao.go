package webacls

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// WebACLDAO provides data access for WAFv2 Web ACLs
type WebACLDAO struct {
	dao.BaseDAO
	client *wafv2.Client
}

// NewWebACLDAO creates a new WebACLDAO
func NewWebACLDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new wafv2/webacls dao")
	}
	return &WebACLDAO{
		BaseDAO: dao.NewBaseDAO("wafv2", "web-acls"),
		client:  wafv2.NewFromConfig(cfg),
	}, nil
}

// List returns all WAFv2 Web ACLs (both REGIONAL and CLOUDFRONT scopes)
func (d *WebACLDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource

	// List REGIONAL Web ACLs
	regionalResources, err := d.listByScope(ctx, types.ScopeRegional)
	if err != nil {
		return nil, apperrors.Wrap(err, "list regional web acls")
	}
	resources = append(resources, regionalResources...)

	// List CLOUDFRONT Web ACLs (only available in us-east-1)
	// We'll try to list CloudFront scope but it may fail if not in us-east-1
	cloudfrontResources, err := d.listByScope(ctx, types.ScopeCloudfront)
	if err != nil {
		// CloudFront scope may fail if not in us-east-1, ignore this error
		// and just return regional resources
		return resources, nil
	}
	resources = append(resources, cloudfrontResources...)

	return resources, nil
}

func (d *WebACLDAO) listByScope(ctx context.Context, scope types.Scope) ([]dao.Resource, error) {
	acls, err := appaws.Paginate(ctx, func(token *string) ([]types.WebACLSummary, *string, error) {
		output, err := d.client.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
			Scope:      scope,
			NextMarker: token,
		})
		if err != nil {
			return nil, nil, err
		}
		return output.WebACLs, output.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(acls))
	for i, acl := range acls {
		resources[i] = NewWebACLResourceFromSummary(acl, scope)
	}

	return resources, nil
}

// Get returns a specific WAFv2 Web ACL by name
// Format: scope/name/id (e.g., "REGIONAL/my-acl/abc123")
func (d *WebACLDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Parse the composite ID (scope/name/id)
	// For simplicity, we'll search through both scopes
	for _, scope := range []types.Scope{types.ScopeRegional, types.ScopeCloudfront} {
		resources, err := d.listByScope(ctx, scope)
		if err != nil {
			continue
		}

		for _, res := range resources {
			if acl, ok := res.(*WebACLResource); ok {
				if acl.GetID() == id || acl.WebACLId() == id {
					// Found the ACL, get full details
					return d.getWebACLDetail(ctx, acl)
				}
			}
		}
	}

	return nil, fmt.Errorf("web acl %s not found", id)
}

func (d *WebACLDAO) getWebACLDetail(ctx context.Context, summary *WebACLResource) (*WebACLResource, error) {
	input := &wafv2.GetWebACLInput{
		Name:  summary.Summary.Name,
		Id:    summary.Summary.Id,
		Scope: summary.Scope,
	}

	output, err := d.client.GetWebACL(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "get web acl")
	}

	return NewWebACLResourceFromDetail(output.WebACL, summary.Scope), nil
}

// Delete deletes a WAFv2 Web ACL
func (d *WebACLDAO) Delete(ctx context.Context, id string) error {
	// First, find the ACL to get its details
	res, err := d.Get(ctx, id)
	if err != nil {
		return err
	}

	acl, ok := res.(*WebACLResource)
	if !ok {
		return fmt.Errorf("invalid resource type")
	}

	// Get lock token
	getInput := &wafv2.GetWebACLInput{
		Name:  acl.Summary.Name,
		Id:    acl.Summary.Id,
		Scope: acl.Scope,
	}

	getOutput, err := d.client.GetWebACL(ctx, getInput)
	if err != nil {
		return apperrors.Wrap(err, "get web acl for lock token")
	}

	// Delete the Web ACL
	deleteInput := &wafv2.DeleteWebACLInput{
		Name:      acl.Summary.Name,
		Id:        acl.Summary.Id,
		Scope:     acl.Scope,
		LockToken: getOutput.LockToken,
	}

	_, err = d.client.DeleteWebACL(ctx, deleteInput)
	if err != nil {
		return apperrors.Wrap(err, "delete web acl")
	}

	return nil
}

// WebACLResource represents a WAFv2 Web ACL
type WebACLResource struct {
	dao.BaseResource
	Summary *types.WebACLSummary
	Detail  *types.WebACL
	Scope   types.Scope
}

// NewWebACLResourceFromSummary creates a new WebACLResource from summary
func NewWebACLResourceFromSummary(summary types.WebACLSummary, scope types.Scope) *WebACLResource {
	id := appaws.Str(summary.Id)
	name := appaws.Str(summary.Name)
	arn := appaws.Str(summary.ARN)

	return &WebACLResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
		Scope:   scope,
	}
}

// NewWebACLResourceFromDetail creates a new WebACLResource from detail
func NewWebACLResourceFromDetail(detail *types.WebACL, scope types.Scope) *WebACLResource {
	id := appaws.Str(detail.Id)
	name := appaws.Str(detail.Name)
	arn := appaws.Str(detail.ARN)

	return &WebACLResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail: detail,
		Scope:  scope,
	}
}

// WebACLName returns the web ACL name
func (r *WebACLResource) WebACLName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Name)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Name)
	}
	return ""
}

// WebACLId returns the web ACL ID
func (r *WebACLResource) WebACLId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Id)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Id)
	}
	return ""
}

// ScopeString returns the scope as string
func (r *WebACLResource) ScopeString() string {
	return string(r.Scope)
}

// RuleCount returns the number of rules
func (r *WebACLResource) RuleCount() int {
	if r.Detail != nil {
		return len(r.Detail.Rules)
	}
	return 0
}

// Rules returns the rules
func (r *WebACLResource) Rules() []types.Rule {
	if r.Detail != nil {
		return r.Detail.Rules
	}
	return nil
}

// DefaultAction returns the default action
func (r *WebACLResource) DefaultAction() string {
	if r.Detail != nil && r.Detail.DefaultAction != nil {
		if r.Detail.DefaultAction.Allow != nil {
			return "ALLOW"
		}
		if r.Detail.DefaultAction.Block != nil {
			return "BLOCK"
		}
	}
	return ""
}

// Description returns the description
func (r *WebACLResource) Description() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Description)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Description)
	}
	return ""
}

// ManagedByFirewallManager returns whether managed by Firewall Manager
func (r *WebACLResource) ManagedByFirewallManager() bool {
	if r.Detail != nil {
		return r.Detail.ManagedByFirewallManager
	}
	return false
}

// Capacity returns the WCU capacity
func (r *WebACLResource) Capacity() int64 {
	if r.Detail != nil {
		return r.Detail.Capacity
	}
	return 0
}

// LabelNamespace returns the label namespace
func (r *WebACLResource) LabelNamespace() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.LabelNamespace)
	}
	return ""
}
