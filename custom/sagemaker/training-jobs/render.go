package trainingjobs

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TrainingJobRenderer renders SageMaker training jobs.
type TrainingJobRenderer struct {
	render.BaseRenderer
}

// NewTrainingJobRenderer creates a new TrainingJobRenderer.
func NewTrainingJobRenderer() render.Renderer {
	return &TrainingJobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sagemaker",
			Resource: "training-jobs",
			Cols: []render.Column{
				{Name: "NAME", Width: 45, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	job, ok := r.(*TrainingJobResource)
	if !ok {
		return ""
	}
	return job.Status()
}

func getAge(r dao.Resource) string {
	job, ok := r.(*TrainingJobResource)
	if !ok {
		return ""
	}
	if t := job.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a training job.
func (r *TrainingJobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*TrainingJobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SageMaker Training Job", job.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", job.GetID())
	d.Field("ARN", job.GetARN())
	d.Field("Status", job.Status())
	if job.GetSecondaryStatus() != "" {
		d.Field("Secondary Status", job.GetSecondaryStatus())
	}
	if job.GetFailureReason() != "" {
		d.Field("Failure Reason", job.GetFailureReason())
	}
	if job.GetTuningJobArn() != "" {
		d.Field("Tuning Job ARN", job.GetTuningJobArn())
	}

	// Algorithm
	if job.GetAlgorithmTrainingImage() != "" || job.GetAlgorithmImage() != "" {
		d.Section("Algorithm")
		if job.GetAlgorithmImage() != "" {
			d.Field("Algorithm", job.GetAlgorithmImage())
		}
		if job.GetAlgorithmTrainingImage() != "" {
			d.Field("Training Image", job.GetAlgorithmTrainingImage())
		}
	}

	// Resources
	if job.GetInstanceType() != "" || job.GetInstanceCount() > 0 {
		d.Section("Resources")
		if job.GetInstanceType() != "" {
			d.Field("Instance Type", job.GetInstanceType())
		}
		if job.GetInstanceCount() > 0 {
			d.Field("Instance Count", fmt.Sprintf("%d", job.GetInstanceCount()))
		}
		if job.GetVolumeSizeInGB() > 0 {
			d.Field("Volume Size", fmt.Sprintf("%d GB", job.GetVolumeSizeInGB()))
		}
		if job.GetEnableSpotTraining() {
			d.Field("Spot Training", "Enabled")
		}
		if job.GetEnableNetworkIsolation() {
			d.Field("Network Isolation", "Enabled")
		}
	}

	// Training Time
	if job.GetTrainingTimeInSeconds() > 0 || job.GetBillableTimeInSeconds() > 0 {
		d.Section("Training Metrics")
		if job.GetTrainingTimeInSeconds() > 0 {
			d.Field("Training Time", formatDuration(job.GetTrainingTimeInSeconds()))
		}
		if job.GetBillableTimeInSeconds() > 0 {
			d.Field("Billable Time", formatDuration(job.GetBillableTimeInSeconds()))
		}
		if job.GetMaxRuntimeSeconds() > 0 {
			d.Field("Max Runtime", formatDuration(job.GetMaxRuntimeSeconds()))
		}
	}

	// Input/Output
	if len(job.GetInputDataConfig()) > 0 || job.GetOutputS3Path() != "" || job.GetModelArtifactsS3() != "" {
		d.Section("Data Configuration")
		for _, ch := range job.GetInputDataConfig() {
			if ch.ChannelName != nil && ch.DataSource != nil && ch.DataSource.S3DataSource != nil {
				d.Field("Input: "+*ch.ChannelName, *ch.DataSource.S3DataSource.S3Uri)
			}
		}
		if job.GetOutputS3Path() != "" {
			d.Field("Output S3", job.GetOutputS3Path())
		}
		if job.GetModelArtifactsS3() != "" {
			d.Field("Model Artifacts", job.GetModelArtifactsS3())
		}
	}

	// Final Metrics
	if metrics := job.GetFinalMetrics(); len(metrics) > 0 {
		d.Section("Final Metrics")
		for _, m := range metrics {
			if m.MetricName != nil && m.Value != nil {
				d.Field(*m.MetricName, fmt.Sprintf("%.4f", *m.Value))
			}
		}
	}

	// Hyperparameters (show first few)
	if hp := job.GetHyperParameters(); len(hp) > 0 {
		d.Section("Hyperparameters")
		count := 0
		for k, v := range hp {
			if count >= 10 {
				d.Field("...", fmt.Sprintf("(%d more)", len(hp)-10))
				break
			}
			d.Field(k, v)
			count++
		}
	}

	// IAM
	if job.GetRoleArn() != "" {
		d.Section("IAM")
		d.Field("Role ARN", job.GetRoleArn())
	}

	// Timestamps
	d.Section("Timestamps")
	if t := job.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.GetTrainingStartTime(); t != nil {
		d.Field("Training Started", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.TrainingEndTime(); t != nil {
		d.Field("Training End", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.GetLastModifiedTime(); t != nil {
		d.Field("Last Modified", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// formatDuration formats seconds into a human-readable duration.
func formatDuration(seconds int32) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
}

// RenderSummary renders summary fields for a training job.
func (r *TrainingJobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*TrainingJobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: job.GetID()},
		{Label: "Status", Value: job.Status()},
	}
}
