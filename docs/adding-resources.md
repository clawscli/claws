# Adding New Resources

This guide explains how to add support for new AWS resources to claws.

## Overview

Adding a new resource requires:
1. Creating DAO (data access) implementation
2. Creating Renderer (UI) implementation
3. Registering in the registry
4. Optionally adding actions in Go code

## Step 1: Create Package Structure

Create a new directory in `custom/<service>/<resource>/`:

```
custom/<service>/<resource>/
├── dao.go       # Data access + Resource type
├── render.go    # UI rendering
├── register.go  # Registry registration
└── actions.go   # (optional) Action executors
```

## Step 2: Implement DAO

`dao.go`:

```go
package myresource

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/service/myservice"
    appaws "github.com/claws/claws/internal/aws"
    "github.com/claws/claws/internal/dao"
)

// DAO implementation
type MyResourceDAO struct {
    dao.BaseDAO
    client *myservice.Client
}

func NewMyResourceDAO(ctx context.Context) (dao.DAO, error) {
    cfg, err := appaws.NewConfig(ctx)
    if err != nil {
        return nil, err
    }
    return &MyResourceDAO{
        BaseDAO: dao.NewBaseDAO("myservice", "myresources"),
        client:  myservice.NewFromConfig(cfg),
    }, nil
}

func (d *MyResourceDAO) List(ctx context.Context) ([]dao.Resource, error) {
    // Check for context filter (for sub-resources or filtered navigation)
    if filter := dao.GetFilterFromContext(ctx, "ParentId"); filter != "" {
        // Filter by parent
    }

    // Use appaws.Paginate for automatic pagination
    items, err := appaws.Paginate(ctx, func(token *string) ([]myservice.Item, *string, error) {
        output, err := d.client.ListItems(ctx, &myservice.ListItemsInput{
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
        resources[i] = NewMyResource(item)
    }
    return resources, nil
}

func (d *MyResourceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
    output, err := d.client.DescribeItem(ctx, &myservice.DescribeItemInput{
        ItemId: &id,
    })
    if err != nil {
        if appaws.IsNotFound(err) {
            return nil, dao.ErrNotFound
        }
        return nil, fmt.Errorf("describe item: %w", err)
    }
    return NewMyResourceWithDetail(output.Item, output), nil
}

func (d *MyResourceDAO) Delete(ctx context.Context, id string) error {
    _, err := d.client.DeleteItem(ctx, &myservice.DeleteItemInput{
        ItemId: &id,
    })
    if err != nil {
        if appaws.IsNotFound(err) {
            return nil
        }
        return fmt.Errorf("delete item: %w", err)
    }
    return nil
}

// Resource type
type MyResource struct {
    dao.BaseResource
    Item   myservice.Item
    Detail *myservice.DescribeOutput // optional, for Get()
}

func NewMyResource(item myservice.Item) *MyResource {
    return &MyResource{
        BaseResource: dao.BaseResource{
            ID:   appaws.Str(item.Id),
            Name: appaws.Str(item.Name),
            ARN:  appaws.Str(item.Arn),
            Data: item,
        },
        Item: item,
    }
}

func NewMyResourceWithDetail(item myservice.Item, detail *myservice.DescribeOutput) *MyResource {
    r := NewMyResource(item)
    r.Detail = detail
    return r
}

// Helper methods for renderer
func (r *MyResource) Status() string {
    return string(r.Item.Status)
}

func (r *MyResource) CreatedAt() string {
    return render.FormatAge(appaws.Time(r.Item.CreatedAt))
}
```

## Step 3: Implement Renderer

`render.go`:

```go
package myresource

import (
    "github.com/claws/claws/internal/dao"
    "github.com/claws/claws/internal/render"
)

// Ensure interface compliance
var _ render.Navigator = (*MyResourceRenderer)(nil)

type MyResourceRenderer struct {
    render.BaseRenderer
}

func NewMyResourceRenderer() render.Renderer {
    return &MyResourceRenderer{
        BaseRenderer: render.BaseRenderer{
            Service:  "myservice",
            Resource: "myresources",
            Cols: []render.Column{
                {
                    Name:  "NAME",
                    Width: 30,
                    Getter: func(r dao.Resource) string {
                        return r.GetName()
                    },
                    Priority: 0,
                },
                {
                    Name:  "STATUS",
                    Width: 12,
                    Getter: func(r dao.Resource) string {
                        if mr, ok := r.(*MyResource); ok {
                            return mr.Status()
                        }
                        return ""
                    },
                    Priority: 1,
                },
                {
                    Name:  "AGE",
                    Width: 10,
                    Getter: func(r dao.Resource) string {
                        if mr, ok := r.(*MyResource); ok {
                            return mr.CreatedAt()
                        }
                        return ""
                    },
                    Priority: 2,
                },
            },
        },
    }
}

func (r *MyResourceRenderer) RenderDetail(resource dao.Resource) string {
    mr, ok := resource.(*MyResource)
    if !ok {
        return ""
    }

    d := render.NewDetailBuilder()
    d.Title("My Resource", mr.GetName())

    d.Section("Basic Information")
    d.Field("ID", mr.GetID())
    d.Field("Name", mr.GetName())
    d.Field("ARN", mr.GetARN())
    d.Field("Status", mr.Status())
    d.Field("Created", mr.CreatedAt())

    // Add more sections as needed
    if mr.Detail != nil {
        d.Section("Configuration")
        d.Field("Setting1", appaws.Str(mr.Detail.Setting1))
        d.Field("Setting2", appaws.Str(mr.Detail.Setting2))
    }

    // For complex objects, use JSON display
    d.Section("Full Details")
    d.JSON(mr.Item)

    return d.String()
}

func (r *MyResourceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
    mr, ok := resource.(*MyResource)
    if !ok {
        return nil
    }

    return []render.SummaryField{
        {Label: "Name", Value: mr.GetName()},
        {Label: "Status", Value: mr.Status()},
        {Label: "Created", Value: mr.CreatedAt()},
    }
}

// IMPORTANT: Method signature must take resource dao.Resource parameter
func (r *MyResourceRenderer) Navigations(resource dao.Resource) []render.Navigation {
    mr, ok := resource.(*MyResource)
    if !ok {
        return nil
    }

    return []render.Navigation{
        {
            Key:         "c",
            Label:       "Children",
            Service:     "myservice",
            Resource:    "children",
            FilterField: "ParentId",      // Field name for DAO filter
            FilterValue: mr.GetID(),      // Actual value from resource
        },
        {
            Key:         "l",
            Label:       "Logs",
            Service:     "cloudwatch",
            Resource:    "log-groups",
            FilterField: "LogGroupName",
            FilterValue: "/aws/myservice/" + mr.GetName(),
        },
    }
}
```

## Step 4: Register

`register.go`:

```go
package myresource

import (
    "context"

    "github.com/claws/claws/internal/dao"
    "github.com/claws/claws/internal/registry"
    "github.com/claws/claws/internal/render"
)

func init() {
    registry.Global.RegisterCustom("myservice", "myresources", registry.Entry{
        DAOFactory: func(ctx context.Context) (dao.DAO, error) {
            return NewMyResourceDAO(ctx)
        },
        RendererFactory: func() render.Renderer {
            return NewMyResourceRenderer()
        },
    })
}
```

## Step 5: Import in main.go

Add import to `cmd/claws/main.go`:

```go
import (
    // ... existing imports
    _ "github.com/claws/claws/custom/myservice/myresources"
)
```

## Step 6: Add Actions (Optional)

Create `actions.go` in the same directory:

```go
package myresource

import (
    "context"
    "fmt"

    "github.com/claws/claws/internal/action"
    appaws "github.com/claws/claws/internal/aws"
    "github.com/claws/claws/internal/dao"
)

func init() {
    action.RegisterExecutor("myservice", "myresources", ExecuteAction)
}

func ExecuteAction(ctx context.Context, act action.Action, resource dao.Resource) error {
    mr := resource.(*MyResource)

    cfg, err := appaws.NewConfig(ctx)
    if err != nil {
        return err
    }
    client := myservice.NewFromConfig(cfg)

    switch act.Name {
    case "Delete":
        _, err := client.DeleteItem(ctx, &myservice.DeleteItemInput{
            ItemId: &mr.ID,
        })
        if err != nil {
            return fmt.Errorf("delete item: %w", err)
        }
    }
    return nil
}
```

## PaginatedDAO (for Large Datasets)

For resources that may return thousands of items (e.g., CloudTrail events), implement `PaginatedDAO`:

```go
type MyResourceDAO struct {
    dao.BaseDAO
    client *myservice.Client
}

// Regular List for standard usage
func (d *MyResourceDAO) List(ctx context.Context) ([]dao.Resource, error) {
    // Return first page only with reasonable limit
    return d.ListPage(ctx, 100, "")
}

// ListPage for manual pagination with 'N' key
func (d *MyResourceDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
    input := &myservice.ListItemsInput{
        MaxResults: aws.Int32(int32(pageSize)),
    }
    if pageToken != "" {
        input.NextToken: aws.String(pageToken),
    }

    output, err := d.client.ListItems(ctx, input)
    if err != nil {
        return nil, "", fmt.Errorf("list items: %w", err)
    }

    resources := make([]dao.Resource, len(output.Items))
    for i, item := range output.Items {
        resources[i] = NewMyResource(item)
    }

    nextToken := ""
    if output.NextToken != nil {
        nextToken = *output.NextToken
    }

    return resources, nextToken, nil
}
```

## Sub-Resources

For resources that are only accessible via navigation (e.g., require parent context):

1. Add to `isSubResource()` in `internal/registry/registry.go`:

```go
func isSubResource(resource string) bool {
    subResources := []string{
        "events", "outputs", "resources", "log-streams", "quotas",
        "mychildren",  // Add your sub-resource here
    }
    return slices.Contains(subResources, resource)
}
```

2. Use context filtering in DAO:

```go
func (d *ChildDAO) List(ctx context.Context) ([]dao.Resource, error) {
    parentId := dao.GetFilterFromContext(ctx, "ParentId")
    if parentId == "" {
        return nil, fmt.Errorf("ParentId filter required")
    }
    // List children for parent
}
```

## Tips

1. **Use `BaseDAO`** - Embed `dao.BaseDAO` for default `ServiceName()` and `ResourceType()` implementations.

2. **Use `BaseRenderer`** - Embed `render.BaseRenderer` for default `RenderRow()` and column handling.

3. **Use `DetailBuilder`** - Use `render.NewDetailBuilder()` for consistent detail views.

4. **Use AWS Helpers** - Always use `appaws.Str()`, `appaws.Int32()`, etc. for safe pointer dereferencing.

5. **Use `appaws.Paginate`** - For List methods, use pagination helper to collect all results.

6. **Filter by Name, Not ARN** - When setting up navigation filters, prefer using names over ARNs for reliability. ARNs can cause issues with client-side filtering.

7. **Sort Results** - If listing from multiple sources, sort results for consistent ordering.

8. **Error Handling** - Use `appaws.IsNotFound()` to check for "not found" errors.

9. **Test Locally**:
   ```bash
   task build && ./claws
   # Navigate to :myservice/myresources
   ```

10. **Check Dead Code**:
    ```bash
    go run golang.org/x/tools/cmd/deadcode@latest ./...
    ```

## Navigation Implementation Checklist

When implementing navigation between resources:

1. **Renderer `Navigations(resource)` method**:
   - Must return `[]render.Navigation` with `FilterField` and `FilterValue` set
   - `FilterValue` must be dynamically extracted from the current resource

2. **Target DAO `List(ctx)` method**:
   - Must call `dao.GetFilterFromContext(ctx, "FilterFieldName")` to get the filter value
   - Must use the filter value in the API call

### Common Mistakes to AVOID

1. **Wrong method signature**:
   - Wrong: `func (r *Renderer) GetNavigations() []render.Navigation`
   - Correct: `func (r *Renderer) Navigations(resource dao.Resource) []render.Navigation`

2. **Static FilterValue**:
   - Wrong: Only setting `FilterField` without `FilterValue`
   - Correct: Extract value from resource and set both

3. **DAO ignoring filter**:
   - The DAO's `List` method MUST check for filters using `dao.GetFilterFromContext()`
