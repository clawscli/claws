# Architecture

This document describes the architecture of claws.

## Overview

claws is built with a modular architecture that separates concerns into distinct layers:

```
┌─────────────────────────────────────────────────────────────┐
│                         TUI Layer                           │
│  (Bubbletea App, Views, Rendering)                         │
├─────────────────────────────────────────────────────────────┤
│                      Registry Layer                         │
│  (Service/Resource registration, alias resolution)          │
├─────────────────────────────────────────────────────────────┤
│                     Business Layer                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    custom/                           │   │
│  │  (Service implementations: DAO + Renderer)          │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                       AWS SDK Layer                         │
│  (AWS SDK for Go v2)                                       │
└─────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
claws/
├── cmd/claws/              # Application entry point
├── internal/
│   ├── app/                # Main Bubbletea application
│   ├── aws/                # AWS client management and helpers
│   │   ├── client.go       # NewConfig() for AWS config loading
│   │   ├── paginate.go     # Paginate(), PaginateIter() helpers
│   │   ├── errors.go       # IsNotFound(), IsAccessDenied(), etc.
│   │   └── pointers.go     # Str(), Int32(), Int64(), Time() helpers
│   ├── action/             # Action framework (API calls, exec commands)
│   ├── config/             # Application configuration (profile, region)
│   ├── dao/                # Data Access Object interface + context filtering
│   ├── registry/           # Service/resource registration + aliases
│   ├── render/             # Renderer interface, DetailBuilder, Navigation
│   ├── ui/                 # Theme system and UI utilities
│   └── view/               # View components (browser, detail, command, help)
├── custom/                 # All 65 service implementations
│   ├── ec2/                # EC2 (13 resources)
│   ├── iam/                # IAM (5 resources)
│   ├── glue/               # Glue (5 resources)
│   ├── bedrock/            # Bedrock (3 resources)
│   ├── bedrockagent/       # Bedrock Agent (5 resources)
│   ├── sagemaker/          # SageMaker (4 resources)
│   └── ...                 # 59 more services
```

## Core Concepts

### DAO (Data Access Object)

The DAO interface provides data access for AWS resources:

```go
type DAO interface {
    ServiceName() string
    ResourceType() string
    List(ctx context.Context) ([]Resource, error)
    Get(ctx context.Context, id string) (Resource, error)
    Delete(ctx context.Context, id string) error
}
```

Each resource type implements this interface. The `BaseDAO` struct provides default implementations for common methods.

**PaginatedDAO**: For large datasets, implement the optional `PaginatedDAO` interface:

```go
type PaginatedDAO interface {
    DAO
    ListPage(ctx context.Context, pageSize int, pageToken string) ([]Resource, string, error)
}
```

**Context Filtering**: DAOs can receive filter parameters via context:

```go
// Set filter in context
ctx = dao.WithFilter(ctx, "VpcId", "vpc-12345")

// Retrieve filter in DAO
vpcId := dao.GetFilterFromContext(ctx, "VpcId")
```

### Renderer

The Renderer interface handles UI rendering:

```go
type Renderer interface {
    Columns() []Column
    RenderRow(resource dao.Resource) table.Row
    RenderDetail(resource dao.Resource) string
    RenderSummary(resource dao.Resource) []SummaryField
}
```

**Navigator**: Optional interface for cross-resource navigation:

```go
type Navigator interface {
    Navigations(resource dao.Resource) []Navigation
}
```

### Registry

The registry manages service/resource registrations:

```go
registry.Global.RegisterCustom("ec2", "instances", registry.Entry{
    DAOFactory:      func(ctx context.Context) (dao.DAO, error) { ... },
    RendererFactory: func() render.Renderer { ... },
})
```

**Service Aliases**: Short names for common services (e.g., `cfn` → `cloudformation`, `sfn` → `stepfunctions`)

**Sub-Resources**: Resources only accessible via navigation (e.g., `cloudformation/events`)

### Actions

Actions define operations that can be performed on resources:

| Type | Description |
|------|-------------|
| `api` | AWS API call (e.g., StopInstances) |
| `exec` | Execute shell command (e.g., SSH) |
| `view` | Navigate to another view |

Actions are defined in Go code within each resource's `actions.go` file:

```go
func init() {
    action.Global.Register("ec2", "instances", []action.Action{
        {Name: "Stop Instance", Shortcut: "S", Type: action.ActionTypeAPI, Confirm: action.ConfirmSimple},
        {Name: "SSH", Shortcut: "s", Type: action.ActionTypeExec},
    })
    action.RegisterExecutor("ec2", "instances", ExecuteAction)
}
```

**ConfirmLevel**: Actions can specify confirmation requirements:

| Level | Description |
|-------|-------------|
| `ConfirmNone` | No confirmation (default) |
| `ConfirmSimple` | Yes/No confirmation |
| `ConfirmDangerous` | Requires typing resource ID (destructive actions) |

### Navigation

Resources can define navigation shortcuts to related resources:

```go
type Navigation struct {
    Key         string        // Shortcut key (e.g., "v")
    Label       string        // Display label (e.g., "VPC")
    Service     string        // Target service
    Resource    string        // Target resource type
    FilterField string        // Filter field name (e.g., "VpcId")
    FilterValue string        // Filter value (extracted from current resource)
    AutoReload  bool          // Auto-refresh (for events, logs)
}
```

## Multi-Region Support

claws supports querying multiple AWS regions simultaneously via the `R` key.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    View Layer                               │
│  ResourceBrowser detects multi-region, spawns goroutines    │
├─────────────────────────────────────────────────────────────┤
│                   Registry Layer                            │
│  GetDAO() auto-wraps with RegionalDAOWrapper                │
├─────────────────────────────────────────────────────────────┤
│                   Wrapper Layer                             │
│  RegionalDAOWrapper / PaginatedDAOWrapper                   │
│  - Wraps resources with region metadata                     │
│  - Preserves concrete types for rendering                   │
├─────────────────────────────────────────────────────────────┤
│                    DAO Layer                                │
│  164 custom DAOs - unmodified, region-agnostic              │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

**RegionalDAOWrapper** (`internal/registry/wrapper.go`):
- Automatically wraps all DAOs when region override is present in context
- `dao.WrapWithRegion(resource, region)` adds region metadata
- `dao.UnwrapResource(resource)` retrieves original for type assertions
- Backward compatible: no wrapping when single-region mode

**Parallel Fetching** (`internal/view/resource_browser.go`):
```go
func (r *ResourceBrowser) fetchMultiRegionResources(regions []string, ...) {
    results := make(chan regionResult, len(regions))
    for _, region := range regions {
        go func(region string) {
            regionCtx := aws.WithRegionOverride(r.ctx, region)
            d, _ := r.registry.GetDAO(regionCtx, service, resourceType)
            resources, _ := d.List(regionCtx)
            results <- regionResult{region, resources, nil}
        }(region)
    }
    // Collect results, handle partial failures
}
```

**Double-Wrap Prevention** (`internal/registry/registry.go`):
```go
// GetDAO checks if delegate is already wrapped
if _, ok := delegate.(*RegionalDAOWrapper); ok {
    return delegate, nil
}
if _, ok := delegate.(*PaginatedDAOWrapper); ok {
    return delegate, nil
}
```

### Resource Flow

1. User selects multiple regions via `R` key
2. `ResourceBrowser.fetchMultiRegionResources()` spawns goroutines per region
3. Each goroutine: `aws.WithRegionOverride(ctx, region)` → `GetDAO()` → `List()`
4. `GetDAO()` auto-wraps DAO with `RegionalDAOWrapper`
5. Wrapper calls underlying DAO, wraps each resource with region
6. Results collected, merged, displayed with Region column
7. Before rendering/actions, `dao.UnwrapResource()` retrieves concrete type

### Partial Failure Handling

If some regions fail (access denied, timeout, etc.):
- Successful regions display normally
- Errors logged at WARN level
- User sees partial results without disruption

## AWS Helper Functions

The `internal/aws/` package provides essential helpers:

### Config Loading
```go
cfg, err := appaws.NewConfig(ctx)  // Load AWS config from environment
```

### Pagination
```go
// Batch pagination - collects all results
items, err := appaws.Paginate(ctx, func(token *string) ([]Item, *string, error) {
    output, err := client.ListItems(ctx, &ListItemsInput{NextToken: token})
    if err != nil {
        return nil, nil, err
    }
    return output.Items, output.NextToken, nil
})

// Streaming pagination - processes items one at a time
for item := range appaws.PaginateIter(ctx, fetchFunc) {
    // Process item
}
```

### Error Handling
```go
if appaws.IsNotFound(err) { }      // Check for "not found" errors
if appaws.IsAccessDenied(err) { }  // Check for "access denied" errors
if appaws.IsThrottling(err) { }    // Check for rate limiting
```

### Safe Pointer Dereferencing
```go
name := appaws.Str(item.Name)        // *string → string
count := appaws.Int32(item.Count)    // *int32 → int32
size := appaws.Int64(item.Size)      // *int64 → int64
created := appaws.Time(item.Created) // *time.Time → time.Time
```

## Theme System

All UI colors are centralized in `internal/ui/theme.go`:

```go
t := ui.Current()           // Get current theme
ui.DimStyle()               // Helper for dim text
ui.SuccessStyle()           // Helper for success color
ui.WarningStyle()           // Helper for warning color
ui.DangerStyle()            // Helper for error color
```

## Views

| View | Description |
|------|-------------|
| Service Browser | List of available AWS services |
| Resource Browser | Table view of resources with filtering and sorting |
| Detail View | Detailed resource information with scrolling |
| Command Mode | `:` command input for navigation and sorting |
| Filter Mode | `/` search input for filtering |
| Help View | `?` key bindings reference (modal) |
| Action Menu | `a` available actions for resource (modal) |
| Region Selector | `R` AWS region switching (modal) |
| Profile Selector | `P` AWS profile switching (modal) |

### Modal System

Some views (Help, Region Selector, Profile Selector, Action Menu) display as modals that overlay the current view rather than pushing to the view stack.

**Key Characteristics:**
- Modals don't affect the view stack (`viewStack` remains unchanged)
- Support nesting via modal stack (e.g., Profile Selector → Profile Detail)
- Dismissed with `esc`, `q`, or `backspace`
- Automatically cleared on region/profile change

**Modal Stack Flow:**
```
┌─────────────────────────────────────────────────────────────┐
│  ShowModalMsg    →  Push current modal to stack, show new   │
│  HideModalMsg    →  Pop stack (restore previous or close)   │
│  NavigateMsg     →  Clear stack, close all modals           │
│  Region/Profile  →  Clear stack, refresh underlying view    │
└─────────────────────────────────────────────────────────────┘
```

**Width Constants** (`internal/view/modal.go`):
- `ModalWidthHelp = 70`
- `ModalWidthRegion = 45`
- `ModalWidthProfile = 55`
- `ModalWidthProfileDetail = 65`
- `ModalWidthActionMenu = 60`

## Configuration

Application configuration is stored in `~/.config/claws/config.yaml`:

```yaml
startup:
  profiles:
    - my-aws-profile
  regions:
    - us-east-1
theme: nord
```

AWS credentials and config are read from standard locations:
- `~/.aws/credentials`
- `~/.aws/config`
- Environment variables

## Performance Optimizations

- **Style Caching**: Lipgloss styles are cached in struct fields to avoid per-frame allocations
- **Lazy Loading**: Resources are loaded on-demand when navigating to a service
- **Pagination**: Large result sets use AWS SDK pagination with `appaws.Paginate`
- **Manual Pagination**: For very large datasets, use `PaginatedDAO` with `N` key for next page

## Logging

Structured logging via `internal/log/`:

```go
log.Debug("operation completed", "duration", elapsed)
log.Info("action executed", "service", svc, "resource", res)
log.Warn("resource not found", "id", id)
log.Error("failed", "error", err)
```

Logs are only written when `-l/--log-file` is specified at startup.
