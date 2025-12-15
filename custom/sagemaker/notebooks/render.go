package notebooks

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// NotebookRenderer renders SageMaker notebook instances.
type NotebookRenderer struct {
	render.BaseRenderer
}

// NewNotebookRenderer creates a new NotebookRenderer.
func NewNotebookRenderer() render.Renderer {
	return &NotebookRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sagemaker",
			Resource: "notebooks",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "INSTANCE TYPE", Width: 18, Getter: getInstanceType},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	notebook, ok := r.(*NotebookResource)
	if !ok {
		return ""
	}
	return notebook.Status()
}

func getInstanceType(r dao.Resource) string {
	notebook, ok := r.(*NotebookResource)
	if !ok {
		return ""
	}
	return notebook.InstanceType()
}

func getAge(r dao.Resource) string {
	notebook, ok := r.(*NotebookResource)
	if !ok {
		return ""
	}
	if t := notebook.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a notebook.
func (r *NotebookRenderer) RenderDetail(resource dao.Resource) string {
	notebook, ok := resource.(*NotebookResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SageMaker Notebook Instance", notebook.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", notebook.GetID())
	d.Field("ARN", notebook.GetARN())
	d.Field("Status", notebook.Status())
	d.Field("Instance Type", notebook.InstanceType())
	if notebook.GetPlatformIdentifier() != "" {
		d.Field("Platform", notebook.GetPlatformIdentifier())
	}
	if notebook.GetURL() != "" {
		d.Field("URL", notebook.GetURL())
	}
	if notebook.GetFailureReason() != "" {
		d.Field("Failure Reason", notebook.GetFailureReason())
	}

	// Storage
	if notebook.GetVolumeSizeInGB() > 0 {
		d.Section("Storage")
		d.Field("Volume Size", fmt.Sprintf("%d GB", notebook.GetVolumeSizeInGB()))
		if notebook.GetKmsKeyId() != "" {
			d.Field("KMS Key", notebook.GetKmsKeyId())
		}
	}

	// IAM
	if notebook.GetRoleArn() != "" {
		d.Section("IAM")
		d.Field("Role ARN", notebook.GetRoleArn())
		d.Field("Root Access", notebook.GetRootAccess())
	}

	// Network
	if notebook.GetSubnetId() != "" || len(notebook.GetSecurityGroups()) > 0 || notebook.GetDirectInternetAccess() != "" {
		d.Section("Network")
		if notebook.GetDirectInternetAccess() != "" {
			d.Field("Internet Access", notebook.GetDirectInternetAccess())
		}
		if notebook.GetSubnetId() != "" {
			d.Field("Subnet", notebook.GetSubnetId())
		}
		if sgs := notebook.GetSecurityGroups(); len(sgs) > 0 {
			d.Field("Security Groups", strings.Join(sgs, ", "))
		}
		if notebook.GetNetworkInterfaceId() != "" {
			d.Field("ENI", notebook.GetNetworkInterfaceId())
		}
	}

	// Lifecycle
	if notebook.GetLifecycleConfigName() != "" {
		d.Section("Lifecycle")
		d.Field("Config", notebook.GetLifecycleConfigName())
	}

	// Git Repositories
	if notebook.GetDefaultCodeRepository() != "" || len(notebook.GetAdditionalCodeRepositories()) > 0 {
		d.Section("Git Repositories")
		if notebook.GetDefaultCodeRepository() != "" {
			d.Field("Default Repository", notebook.GetDefaultCodeRepository())
		}
		for i, repo := range notebook.GetAdditionalCodeRepositories() {
			d.Field(fmt.Sprintf("Additional Repo %d", i+1), repo)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if t := notebook.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := notebook.LastModifiedAt(); t != nil {
		d.Field("Last Modified", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a notebook.
func (r *NotebookRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	notebook, ok := resource.(*NotebookResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: notebook.GetID()},
		{Label: "Status", Value: notebook.Status()},
		{Label: "Instance Type", Value: notebook.InstanceType()},
	}
}
