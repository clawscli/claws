package jobqueues

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobQueueRenderer renders Batch job queues.
// Ensure JobQueueRenderer implements render.Navigator
var _ render.Navigator = (*JobQueueRenderer)(nil)

type JobQueueRenderer struct {
	render.BaseRenderer
}

// NewJobQueueRenderer creates a new JobQueueRenderer.
func NewJobQueueRenderer() render.Renderer {
	return &JobQueueRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "batch",
			Resource: "job-queues",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "PRIORITY", Width: 10, Getter: getPriority},
			},
		},
	}
}

func getState(r dao.Resource) string {
	queue, ok := r.(*JobQueueResource)
	if !ok {
		return ""
	}
	return queue.State()
}

func getStatus(r dao.Resource) string {
	queue, ok := r.(*JobQueueResource)
	if !ok {
		return ""
	}
	return queue.Status()
}

func getPriority(r dao.Resource) string {
	queue, ok := r.(*JobQueueResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", queue.Priority())
}

// RenderDetail renders the detail view for a job queue.
func (r *JobQueueRenderer) RenderDetail(resource dao.Resource) string {
	queue, ok := resource.(*JobQueueResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Batch Job Queue", queue.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", queue.GetID())
	d.Field("ARN", queue.GetARN())
	d.Field("State", queue.State())
	d.Field("Status", queue.Status())
	d.Field("Priority", fmt.Sprintf("%d", queue.Priority()))

	// Scheduling
	if queue.SchedulingPolicy() != "" {
		d.Section("Scheduling")
		d.Field("Scheduling Policy", queue.SchedulingPolicy())
	}

	// Compute Environments
	if queue.Queue != nil && len(queue.Queue.ComputeEnvironmentOrder) > 0 {
		d.Section("Compute Environments")
		for i, ce := range queue.Queue.ComputeEnvironmentOrder {
			label := fmt.Sprintf("Environment %d", i+1)
			d.Field(label, fmt.Sprintf("%s (order: %d)", appaws.Str(ce.ComputeEnvironment), ce.Order))
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a job queue.
func (r *JobQueueRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	queue, ok := resource.(*JobQueueResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: queue.GetID()},
		{Label: "State", Value: queue.State()},
		{Label: "Status", Value: queue.Status()},
		{Label: "Priority", Value: fmt.Sprintf("%d", queue.Priority())},
	}
}

// Navigations returns available navigations from a job queue.
func (r *JobQueueRenderer) Navigations(resource dao.Resource) []render.Navigation {
	queue, ok := resource.(*JobQueueResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "j",
			Label:       "Jobs",
			Service:     "batch",
			Resource:    "jobs",
			FilterField: "JobQueue",
			FilterValue: queue.GetARN(),
		},
	}
}
