package jobs

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobRenderer renders Transcribe jobs.
type JobRenderer struct {
	render.BaseRenderer
}

// NewJobRenderer creates a new JobRenderer.
func NewJobRenderer() render.Renderer {
	return &JobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "transcribe",
			Resource: "jobs",
			Cols: []render.Column{
				{Name: "JOB NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "LANGUAGE", Width: 10, Getter: getLanguage},
				{Name: "OUTPUT TYPE", Width: 15, Getter: getOutputType},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.Status()
}

func getLanguage(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.LanguageCode()
}

func getOutputType(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.OutputLocationType()
}

func getCreated(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	if t := job.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Transcribe job.
func (r *JobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*JobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Transcribe Job", job.JobName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job Name", job.JobName())
	d.Field("Status", job.Status())
	d.Field("Language Code", job.LanguageCode())
	if job.IdentifyLanguage() {
		d.Field("Language Identification", "Enabled")
		if score := job.IdentifiedLanguageScore(); score > 0 {
			d.Field("Language Confidence", fmt.Sprintf("%.2f%%", score*100))
		}
	}
	if job.IdentifyMultipleLanguages() {
		d.Field("Multiple Language ID", "Enabled")
	}

	// Media
	d.Section("Media")
	if format := job.MediaFormat(); format != "" {
		d.Field("Format", format)
	}
	if rate := job.MediaSampleRateHertz(); rate > 0 {
		d.Field("Sample Rate", fmt.Sprintf("%d Hz", rate))
	}
	if uri := job.MediaFileUri(); uri != "" {
		d.Field("Media File URI", uri)
	}

	// Output
	d.Section("Output")
	if output := job.OutputLocationType(); output != "" {
		d.Field("Location Type", output)
	}
	if uri := job.TranscriptFileUri(); uri != "" {
		d.Field("Transcript URI", uri)
	}

	// Features
	d.Section("Features")
	if job.ContentRedactionEnabled() {
		d.Field("Content Redaction", "Enabled")
	}
	if job.SubtitlesEnabled() {
		d.Field("Subtitles", "Enabled")
		if formats := job.SubtitleFormats(); len(formats) > 0 {
			d.Field("Subtitle Formats", strings.Join(formats, ", "))
		}
	}
	if model := job.ModelName(); model != "" {
		d.Field("Custom Model", model)
	}

	// Failure
	if reason := job.FailureReason(); reason != "" {
		d.Section("Failure")
		d.Field("Reason", reason)
	}

	// Timestamps
	d.Section("Timestamps")
	if t := job.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.StartTime(); t != nil {
		d.Field("Started", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.CompletionTime(); t != nil {
		d.Field("Completed", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Transcribe job.
func (r *JobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*JobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Job Name", Value: job.JobName()},
		{Label: "Status", Value: job.Status()},
		{Label: "Language", Value: job.LanguageCode()},
	}

	if output := job.OutputLocationType(); output != "" {
		fields = append(fields, render.SummaryField{Label: "Output Type", Value: output})
	}

	if reason := job.FailureReason(); reason != "" {
		fields = append(fields, render.SummaryField{Label: "Failure Reason", Value: reason})
	}

	return fields
}
