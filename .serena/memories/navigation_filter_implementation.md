# Navigation Filter Implementation Pattern

## CRITICAL: When implementing navigation between resources

### Required Components

1. **Renderer `Navigations(resource)` method**:
   - Must return `[]render.Navigation` with `FilterField` and `FilterValue` set
   - `FilterValue` must be dynamically extracted from the current resource

2. **DAO `List(ctx)` method**:
   - Must call `dao.GetFilterFromContext(ctx, "FilterFieldName")` to get the filter value
   - Must use the filter value in the API call

### Common Mistakes to AVOID

1. **Using `GetNavigations()` instead of `Navigations(resource dao.Resource)`**
   - Wrong: `func (r *Renderer) GetNavigations() []render.Navigation`
   - Correct: `func (r *Renderer) Navigations(resource dao.Resource) []render.Navigation`

2. **Forgetting to set `FilterValue` dynamically**
   - Wrong: Only setting `FilterField` without `FilterValue`
   - Correct: Extract value from resource and set both `FilterField` and `FilterValue`

3. **DAO not processing the filter**
   - The DAO's `List` method MUST check for filters using `dao.GetFilterFromContext()`
   - If filter is provided, include it in the AWS API call

### Example Implementation

```go
// Renderer - Navigations method
func (r *TargetGroupRenderer) Navigations(resource dao.Resource) []render.Navigation {
    rr, ok := resource.(*TargetGroupResource)
    if !ok {
        return nil
    }
    
    return []render.Navigation{
        {
            Key:         "l",
            Label:       "Load Balancer",
            Service:     "elbv2",
            Resource:    "load-balancers",
            FilterField: "LoadBalancerArn",      // Field name for DAO filter
            FilterValue: rr.LoadBalancerArns()[0], // Actual value from resource
        },
    }
}

// DAO - List method with filter support
func (d *LoadBalancerDAO) List(ctx context.Context) ([]dao.Resource, error) {
    // MUST check for filter
    lbArn := dao.GetFilterFromContext(ctx, "LoadBalancerArn")
    
    input := &elasticloadbalancingv2.DescribeLoadBalancersInput{}
    
    // MUST use filter in API call if provided
    if lbArn != "" {
        input.LoadBalancerArns = []string{lbArn}
    }
    
    // ... rest of implementation
}
```

### Debugging Checklist

When navigation doesn't work:
1. Check that `Navigations()` method signature is correct (takes `resource dao.Resource`)
2. Check that `FilterValue` is being set (not empty)
3. Check that DAO's `List()` method calls `dao.GetFilterFromContext()` for the filter field
4. Check that DAO uses the filter value in the API call

### CRITICAL: Child-to-Parent Navigation Bug (FIXED 2024-12)

**Problem**: `matchesFieldFilter` in `resource_browser_filter.go` was extracting resource name from ARN
BEFORE comparing with resource ID. This broke child→parent navigation for resources where ID is the full ARN.

**Root Cause**: Line 242-244 extracted resource name from ARN first:
```go
if strings.HasPrefix(filterValue, "arn:aws:") {
    filterValue = appaws.ExtractResourceName(filterValue)  // WRONG ORDER!
}
```

**Fix**: Compare with original ARN FIRST, then try extracted name:
```go
// First, try matching by ID or Name with the original filter value
if res.GetID() == filterValue || res.GetName() == filterValue {
    return true
}
// Then try extracted name for IAM-style resources
if strings.HasPrefix(filterValue, "arn:aws:") {
    extractedName := appaws.ExtractResourceName(filterValue)
    if res.GetID() == extractedName || res.GetName() == extractedName {
        return true
    }
}
```

**Affected**: Any child→parent navigation where parent's ID is a full ARN (LoadBalancer, StateMachine, etc.)

### Reference Files

- `internal/view/resource_browser.go:172-174` - Where filter is added to context
- `internal/dao/dao.go:97-110` - WithFilter and GetFilterFromContext functions
- `internal/view/view.go:115-122` - Where navigation creates filtered browser
