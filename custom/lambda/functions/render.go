package functions

import (
	"fmt"
	"strings"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FunctionRenderer renders Lambda functions
// Ensure FunctionRenderer implements render.Navigator
var _ render.Navigator = (*FunctionRenderer)(nil)

type FunctionRenderer struct {
	render.BaseRenderer
}

// NewFunctionRenderer creates a new FunctionRenderer
func NewFunctionRenderer() render.Renderer {
	return &FunctionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "lambda",
			Resource: "functions",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetName() }, Priority: 0},
				{Name: "RUNTIME", Width: 15, Getter: getRuntimeDisplay, Priority: 1},
				{Name: "STATE", Width: 10, Getter: getState, Priority: 2},
				{Name: "MEMORY", Width: 8, Getter: getMemory, Priority: 3},
				{Name: "TIMEOUT", Width: 8, Getter: getTimeout, Priority: 4},
				{Name: "SIZE", Width: 10, Getter: getCodeSize, Priority: 5},
				{Name: "MODIFIED", Width: 12, Getter: getModified, Priority: 6},
			},
		},
	}
}

func getRuntimeDisplay(r dao.Resource) string {
	if fn, ok := r.(*FunctionResource); ok {
		runtime := fn.Runtime()
		// Shorten runtime names for display
		switch {
		case strings.HasPrefix(runtime, "python"):
			return strings.Replace(runtime, "python", "py", 1)
		case strings.HasPrefix(runtime, "nodejs"):
			return strings.Replace(runtime, "nodejs", "node", 1)
		case runtime == "provided.al2023":
			return "al2023"
		case runtime == "provided.al2":
			return "al2"
		default:
			return runtime
		}
	}
	return ""
}

func getState(r dao.Resource) string {
	if fn, ok := r.(*FunctionResource); ok {
		state := fn.State()
		switch state {
		case "Active":
			return "active"
		case "Pending":
			return "pending"
		case "Inactive":
			return "stopped"
		case "Failed":
			return "failed"
		default:
			return strings.ToLower(state)
		}
	}
	return ""
}

func getMemory(r dao.Resource) string {
	if fn, ok := r.(*FunctionResource); ok {
		return fmt.Sprintf("%dMB", fn.MemorySize())
	}
	return ""
}

func getTimeout(r dao.Resource) string {
	if fn, ok := r.(*FunctionResource); ok {
		return fmt.Sprintf("%ds", fn.Timeout())
	}
	return ""
}

func getCodeSize(r dao.Resource) string {
	if fn, ok := r.(*FunctionResource); ok {
		return render.FormatSize(fn.CodeSize())
	}
	return ""
}

func getModified(r dao.Resource) string {
	if fn, ok := r.(*FunctionResource); ok {
		lastMod := fn.LastModified()
		if lastMod == "" {
			return ""
		}
		// Parse ISO 8601 format
		t, err := time.Parse("2006-01-02T15:04:05.000+0000", lastMod)
		if err != nil {
			// Try alternative format
			t, err = time.Parse(time.RFC3339, lastMod)
			if err != nil {
				return lastMod[:10] // Return date portion
			}
		}
		return render.FormatAge(t)
	}
	return ""
}

// RenderDetail renders detailed function information
func (r *FunctionRenderer) RenderDetail(resource dao.Resource) string {
	fn, ok := resource.(*FunctionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Lambda Function", fn.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", fn.GetName())
	d.Field("ARN", fn.GetARN())
	d.FieldStyled("State", fn.State(), render.StateColorer()(strings.ToLower(fn.State())))
	d.Field("Runtime", fn.Runtime())
	d.Field("Package Type", fn.PackageType())

	if handler := fn.Handler(); handler != "" {
		d.Field("Handler", handler)
	}

	if desc := fn.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Function URL (if exists)
	if fn.FunctionURL != "" {
		d.Section("Function URL")
		d.Field("URL", fn.FunctionURL)
	}

	// Configuration
	d.Section("Configuration")
	d.Field("Memory", fmt.Sprintf("%d MB", fn.MemorySize()))
	d.Field("Timeout", fmt.Sprintf("%d seconds", fn.Timeout()))
	d.Field("Ephemeral Storage", fmt.Sprintf("%d MB", fn.EphemeralStorageSize()))
	d.Field("Code Size", render.FormatSize(fn.CodeSize()))

	// Concurrency
	if fn.ReservedConcurrency != nil {
		d.Field("Reserved Concurrency", fmt.Sprintf("%d", *fn.ReservedConcurrency))
	}

	if archs := fn.Architectures(); len(archs) > 0 {
		var archStrs []string
		for _, arch := range archs {
			archStrs = append(archStrs, string(arch))
		}
		d.Field("Architecture", strings.Join(archStrs, ", "))
	}

	// SnapStart
	if snap := fn.SnapStart(); snap != nil && snap.ApplyOn != "" {
		d.Field("SnapStart", string(snap.ApplyOn))
		if snap.OptimizationStatus != "" {
			d.Field("SnapStart Status", string(snap.OptimizationStatus))
		}
	}

	// Tracing
	if tracing := fn.TracingConfig(); tracing != "" {
		d.Field("X-Ray Tracing", tracing)
	}

	// IAM
	if role := fn.Role(); role != "" {
		d.Section("IAM")
		d.Field("Role Name", appaws.ExtractResourceName(role))
		d.Field("Role ARN", role)
	}

	// Environment variables (just count, not values for security)
	if env := fn.Item.Environment; env != nil && len(env.Variables) > 0 {
		d.Section("Environment")
		d.Field("Variables", fmt.Sprintf("%d defined", len(env.Variables)))
	}

	// Dead Letter Queue
	if dlq := fn.DeadLetterConfig(); dlq != nil && dlq.TargetArn != nil && *dlq.TargetArn != "" {
		d.Section("Dead Letter Queue")
		d.Field("Target ARN", *dlq.TargetArn)
	}

	// Layers
	if layers := fn.Layers(); len(layers) > 0 {
		d.Section("Layers")
		for _, layer := range layers {
			if layer.Arn != nil {
				d.Field("", appaws.ExtractResourceName(*layer.Arn))
			}
		}
	}

	// Encryption
	if kmsKey := fn.KMSKeyArn(); kmsKey != "" {
		d.Section("Encryption")
		d.Field("KMS Key ARN", kmsKey)
	}

	// VPC config
	if vpc := fn.Item.VpcConfig; vpc != nil && vpc.VpcId != nil && *vpc.VpcId != "" {
		d.Section("VPC Configuration")
		d.Field("VPC ID", *vpc.VpcId)
		if len(vpc.SubnetIds) > 0 {
			d.Field("Subnets", strings.Join(vpc.SubnetIds, ", "))
		}
		if len(vpc.SecurityGroupIds) > 0 {
			d.Field("Security Groups", strings.Join(vpc.SecurityGroupIds, ", "))
		}
	}

	// File System (EFS) Configuration
	if fsConfigs := fn.Item.FileSystemConfigs; len(fsConfigs) > 0 {
		d.Section("File System (EFS)")
		for i, fsConfig := range fsConfigs {
			if fsConfig.Arn != nil {
				d.Field(fmt.Sprintf("EFS %d ARN", i+1), *fsConfig.Arn)
			}
			if fsConfig.LocalMountPath != nil {
				d.Field(fmt.Sprintf("EFS %d Mount Path", i+1), *fsConfig.LocalMountPath)
			}
		}
	}

	// State reason (for troubleshooting)
	if reason := fn.StateReason(); reason != "" {
		d.Section("State Information")
		d.Field("State Reason", reason)
		if code := fn.StateReasonCode(); code != "" {
			d.Field("Reason Code", code)
		}
	}

	// Timestamps & Version Info
	d.Section("Version Information")
	if lastMod := fn.LastModified(); lastMod != "" {
		d.Field("Last Modified", lastMod)
	}
	if version := fn.Version(); version != "" && version != "$LATEST" {
		d.Field("Version", version)
	}
	if sha := fn.CodeSha256(); sha != "" {
		d.Field("Code SHA256", sha[:16]+"...") // Truncate for display
	}

	// Tags
	d.Tags(fn.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *FunctionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	fn, ok := resource.(*FunctionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: fn.GetName()},
		{Label: "ARN", Value: fn.GetARN()},
		{Label: "Runtime", Value: fn.Runtime()},
		{Label: "State", Value: fn.State()},
		{Label: "Memory", Value: fmt.Sprintf("%d MB", fn.MemorySize())},
		{Label: "Timeout", Value: fmt.Sprintf("%d seconds", fn.Timeout())},
		{Label: "Package", Value: fn.PackageType()},
	}

	if handler := fn.Handler(); handler != "" {
		fields = append(fields, render.SummaryField{Label: "Handler", Value: handler})
	}

	if desc := fn.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if role := fn.Role(); role != "" {
		fields = append(fields, render.SummaryField{Label: "Role", Value: appaws.ExtractResourceName(role)})
	}

	if archs := fn.Architectures(); len(archs) > 0 {
		var archStrs []string
		for _, arch := range archs {
			archStrs = append(archStrs, string(arch))
		}
		fields = append(fields, render.SummaryField{Label: "Architecture", Value: strings.Join(archStrs, ", ")})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *FunctionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	fn, ok := resource.(*FunctionResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to CloudWatch Logs
	logGroupName := "/aws/lambda/" + fn.GetName()
	navs = append(navs, render.Navigation{
		Key:         "l",
		Label:       "Logs",
		Service:     "cloudwatch",
		Resource:    "log-groups",
		FilterField: "LogGroupPrefix",
		FilterValue: logGroupName,
	})

	// Navigate to IAM role
	if role := fn.Role(); role != "" {
		roleName := appaws.ExtractResourceName(role)
		navs = append(navs, render.Navigation{
			Key:         "r",
			Label:       "Role",
			Service:     "iam",
			Resource:    "roles",
			FilterField: "RoleName",
			FilterValue: roleName,
		})
	}

	// VPC navigation (if function is in VPC)
	if fn.Item.VpcConfig != nil && fn.Item.VpcConfig.VpcId != nil && *fn.Item.VpcConfig.VpcId != "" {
		navs = append(navs, render.Navigation{
			Key:         "v",
			Label:       "VPC",
			Service:     "vpc",
			Resource:    "vpcs",
			FilterField: "VpcId",
			FilterValue: *fn.Item.VpcConfig.VpcId,
		})

		// Security Groups navigation
		if len(fn.Item.VpcConfig.SecurityGroupIds) > 0 {
			navs = append(navs, render.Navigation{
				Key:         "g",
				Label:       "Security Groups",
				Service:     "ec2",
				Resource:    "security-groups",
				FilterField: "GroupId",
				FilterValue: fn.Item.VpcConfig.SecurityGroupIds[0],
			})
		}
	}

	return navs
}
