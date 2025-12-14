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
│   │   ├── context.go      # WithAPITimeout()
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

**Service Aliases**: Short names for common services (e.g., `cfn` → `cloudformation`, `cf` → `cloudfront`)

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
        {Name: "Stop Instance", Shortcut: "S", Type: action.ActionTypeAPI, Confirm: true, Dangerous: true},
        {Name: "SSH", Shortcut: "s", Type: action.ActionTypeExec},
    })
    action.RegisterExecutor("ec2", "instances", ExecuteAction)
}
```

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

### Context Timeout
```go
ctx = appaws.WithAPITimeout(ctx)  // Add 30s timeout
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
| Help View | `?` key bindings reference |
| Action Menu | `a` available actions for resource |
| Region Selector | `R` AWS region switching |
| Profile Selector | `P` AWS profile switching |

## Configuration

Application configuration is stored in `~/.config/claws/config.yaml`:

```yaml
profile: my-aws-profile
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
