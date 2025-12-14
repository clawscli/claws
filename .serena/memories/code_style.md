# Code Style and Conventions for claws

## Go Style
- CamelCase for private, PascalCase for exported
- Interfaces end with -er (Renderer, Navigator)
- One package per directory

## Architecture Patterns

### DAO (Data Access Object)
- Implements: `List(ctx)`, `Get(ctx, id)`, `Delete(ctx, id)`
- Embed `dao.BaseDAO` for default implementations
- Use `appaws.Paginate` for pagination
- Use `dao.GetFilterFromContext(ctx, field)` for context filtering

### Renderer
- Implements: `Columns()`, `RenderRow()`, `RenderDetail()`, `RenderSummary()`
- Embed `render.BaseRenderer` for default row rendering
- Use `render.NewDetailBuilder()` for consistent detail views

### Navigator (Optional Interface)
- Implements: `Navigations(resource dao.Resource) []render.Navigation`
- Returns navigation shortcuts for cross-resource navigation
- **CRITICAL**: Method takes `resource dao.Resource` parameter, NOT `GetNavigations()`

### Registry
- Register in `init()` function of `register.go`
- Use `registry.Global.RegisterCustom(service, resource, entry)`

## DRY Helpers (ALWAYS Use These!)

### AWS Helpers (internal/aws/)
```go
cfg, err := appaws.NewConfig(ctx)           // AWS config loading
items, err := appaws.Paginate(ctx, fetchFn) // Batch pagination
for item := range appaws.PaginateIter(ctx, fn) { }  // Streaming pagination
if appaws.IsNotFound(err) { }               // Check "not found" error
if appaws.IsAccessDenied(err) { }           // Check "access denied"
if appaws.IsThrottling(err) { }             // Check rate limiting
ctx = appaws.WithAPITimeout(ctx)            // Add 30s timeout
name := appaws.Str(ptr.Name)                // Safe pointer deref
count := appaws.Int32(ptr.Count)            // Safe int32 deref
```

### Render Helpers (internal/render/)
```go
age := render.FormatAge(createdAt)          // Human-readable age
d := render.NewDetailBuilder()              // Detail view builder
d.Title("Resource", name)
d.Section("Basic Info")
d.Field("ID", id)
```

### DAO Helpers (internal/dao/)
```go
ctx = dao.WithFilter(ctx, "VpcId", vpcId)   // Set filter
filter := dao.GetFilterFromContext(ctx, "VpcId")  // Get filter
```

### Logging (internal/log/)
```go
log.Debug("operation completed", "duration", elapsed)
log.Info("action executed", "service", svc)
log.Warn("resource not found", "id", id)
log.Error("failed", "error", err)
```

## Pagination Pattern
```go
func (d *MyDAO) List(ctx context.Context) ([]dao.Resource, error) {
    items, err := appaws.Paginate(ctx, func(token *string) ([]Item, *string, error) {
        output, err := d.client.ListItems(ctx, &service.ListItemsInput{
            NextToken: token,
        })
        if err != nil {
            return nil, nil, fmt.Errorf("list items: %w", err)
        }
        return output.Items, output.NextToken, nil
    })
    if err != nil {
        return nil, err
    }
    
    resources := make([]dao.Resource, len(items))
    for i, item := range items {
        resources[i] = NewResource(item)
    }
    return resources, nil
}
```

## PaginatedDAO Pattern (for large datasets)
```go
type MyDAO struct {
    dao.BaseDAO
    client *service.Client
}

func (d *MyDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
    input := &service.ListInput{
        MaxResults: aws.Int32(int32(pageSize)),
    }
    if pageToken != "" {
        input.NextToken = aws.String(pageToken)
    }
    // ... implementation
}
```

## Navigation Implementation
```go
func (r *Renderer) Navigations(resource dao.Resource) []render.Navigation {
    res, ok := resource.(*MyResource)
    if !ok {
        return nil
    }
    return []render.Navigation{
        {
            Key:         "v",
            Label:       "VPC",
            Service:     "ec2",
            Resource:    "vpcs",
            FilterField: "VpcId",      // Field name for DAO filter
            FilterValue: res.VpcId(),  // Actual value from resource
        },
    }
}
```

## Style Caching (for Bubbletea Views)
```go
type myViewStyles struct {
    title lipgloss.Style
    label lipgloss.Style
}

func newMyViewStyles() myViewStyles {
    t := ui.Current()
    return myViewStyles{
        title: lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
        label: lipgloss.NewStyle().Foreground(t.TextDim),
    }
}

type MyView struct {
    styles myViewStyles  // Cache in struct
}
```

## Sub-Resources
Resources requiring parent context (e.g., events, quotas, log-streams):
1. Add to `isSubResource()` in `internal/registry/registry.go`
2. Use `dao.GetFilterFromContext()` in DAO to get parent filter
3. Return error if filter is missing

## Adding New Resources
1. Create `custom/<service>/<resource>/` directory
2. Create `dao.go` - DAO + Resource type
3. Create `render.go` - Renderer with columns, detail, summary
4. Create `register.go` - Registry registration in `init()`
5. Import in `cmd/claws/main.go`
6. Optional: Create `actions.go` for action executors
