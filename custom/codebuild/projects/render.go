package projects

import (
	"fmt"
	"sort"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ProjectRenderer renders CodeBuild projects
// Ensure ProjectRenderer implements render.Navigator
var _ render.Navigator = (*ProjectRenderer)(nil)

type ProjectRenderer struct {
	render.BaseRenderer
}

// NewProjectRenderer creates a new ProjectRenderer
func NewProjectRenderer() *ProjectRenderer {
	return &ProjectRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "codebuild",
			Resource: "projects",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "SOURCE", Width: 15, Getter: getSourceType},
				{Name: "ENV TYPE", Width: 15, Getter: getEnvType},
				{Name: "COMPUTE", Width: 20, Getter: getCompute},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if p, ok := r.(*ProjectResource); ok {
		return p.ProjectName()
	}
	return ""
}

func getSourceType(r dao.Resource) string {
	if p, ok := r.(*ProjectResource); ok {
		return p.SourceType()
	}
	return ""
}

func getEnvType(r dao.Resource) string {
	if p, ok := r.(*ProjectResource); ok {
		return p.EnvironmentType()
	}
	return ""
}

func getCompute(r dao.Resource) string {
	if p, ok := r.(*ProjectResource); ok {
		return p.ComputeType()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if p, ok := r.(*ProjectResource); ok {
		if t := p.CreatedAtTime(); t != nil {
			return render.FormatAge(*t)
		}
	}
	return "-"
}

// RenderDetail renders detailed project information
func (r *ProjectRenderer) RenderDetail(resource dao.Resource) string {
	project, ok := resource.(*ProjectResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CodeBuild Project", project.ProjectName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", project.ProjectName())
	if arn := project.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	if desc := project.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Source
	d.Section("Source")
	d.Field("Type", project.SourceType())
	if loc := project.SourceLocation(); loc != "" {
		d.Field("Location", loc)
	}

	// Environment
	d.Section("Environment")
	d.Field("Type", project.EnvironmentType())
	d.Field("Compute Type", project.ComputeType())
	if img := project.EnvironmentImage(); img != "" {
		d.Field("Image", img)
	}

	// Build Settings
	d.Section("Build Settings")
	if timeout := project.TimeoutInMinutes(); timeout > 0 {
		d.Field("Timeout", fmt.Sprintf("%d minutes", timeout))
	}
	if limit := project.ConcurrentBuildLimit(); limit > 0 {
		d.Field("Concurrent Build Limit", fmt.Sprintf("%d", limit))
	}
	d.Field("Badge Enabled", fmt.Sprintf("%v", project.BadgeEnabled()))
	d.Field("Batch Builds Enabled", fmt.Sprintf("%v", project.BatchBuildEnabled()))

	// Service Role
	if role := project.ServiceRole(); role != "" {
		d.Section("IAM")
		d.Field("Service Role", role)
	}

	// Artifacts
	if project.Project.Artifacts != nil {
		artifacts := project.Project.Artifacts
		d.Section("Artifacts")
		d.Field("Type", string(artifacts.Type))
		if artifacts.Location != nil && *artifacts.Location != "" {
			d.Field("Location", *artifacts.Location)
		}
		if artifacts.Packaging != "" {
			d.Field("Packaging", string(artifacts.Packaging))
		}
	}

	// Cache
	if project.Project.Cache != nil && project.Project.Cache.Type != "NO_CACHE" {
		cache := project.Project.Cache
		d.Section("Cache")
		d.Field("Type", string(cache.Type))
		if cache.Location != nil {
			d.Field("Location", *cache.Location)
		}
	}

	// VPC Configuration
	if project.Project.VpcConfig != nil && project.Project.VpcConfig.VpcId != nil && *project.Project.VpcConfig.VpcId != "" {
		vpc := project.Project.VpcConfig
		d.Section("VPC Configuration")
		d.Field("VPC ID", *vpc.VpcId)
		if len(vpc.Subnets) > 0 {
			d.Field("Subnets", fmt.Sprintf("%d configured", len(vpc.Subnets)))
		}
		if len(vpc.SecurityGroupIds) > 0 {
			d.Field("Security Groups", fmt.Sprintf("%d configured", len(vpc.SecurityGroupIds)))
		}
	}

	// Logs
	if project.Project.LogsConfig != nil {
		logs := project.Project.LogsConfig
		d.Section("Logging")
		if logs.CloudWatchLogs != nil && logs.CloudWatchLogs.Status == "ENABLED" {
			d.FieldStyled("CloudWatch Logs", "Enabled", render.SuccessStyle())
			if logs.CloudWatchLogs.GroupName != nil {
				d.Field("  Log Group", *logs.CloudWatchLogs.GroupName)
			}
		}
		if logs.S3Logs != nil && logs.S3Logs.Status == "ENABLED" {
			d.FieldStyled("S3 Logs", "Enabled", render.SuccessStyle())
			if logs.S3Logs.Location != nil {
				d.Field("  Location", *logs.S3Logs.Location)
			}
		}
	}

	// Encryption
	if project.Project.EncryptionKey != nil && *project.Project.EncryptionKey != "" {
		d.Section("Encryption")
		d.Field("KMS Key", *project.Project.EncryptionKey)
	}

	// Webhook
	if project.Project.Webhook != nil {
		d.Section("Webhook")
		if project.Project.Webhook.Url != nil {
			d.Field("URL", *project.Project.Webhook.Url)
		}
	}

	// Tags
	if len(project.Tags) > 0 {
		d.Section("Tags")
		keys := make([]string, 0, len(project.Tags))
		for k := range project.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			d.Field(k, project.Tags[k])
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := project.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if modified := project.LastModified(); modified != "" {
		d.Field("Last Modified", modified)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ProjectRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	project, ok := resource.(*ProjectResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: project.ProjectName()},
		{Label: "Source", Value: project.SourceType()},
		{Label: "Environment", Value: project.EnvironmentType()},
		{Label: "Compute", Value: project.ComputeType()},
	}

	if arn := project.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if timeout := project.TimeoutInMinutes(); timeout > 0 {
		fields = append(fields, render.SummaryField{Label: "Timeout", Value: fmt.Sprintf("%d min", timeout)})
	}

	if created := project.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ProjectRenderer) Navigations(resource dao.Resource) []render.Navigation {
	project, ok := resource.(*ProjectResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "b", Label: "Builds", Service: "codebuild", Resource: "builds",
			FilterField: "ProjectName", FilterValue: project.ProjectName(),
		},
	}
}
