package jobs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transcribe/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// JobDAO provides data access for Transcribe jobs.
type JobDAO struct {
	dao.BaseDAO
	client *transcribe.Client
}

// NewJobDAO creates a new JobDAO.
func NewJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &JobDAO{
		BaseDAO: dao.NewBaseDAO("transcribe", "jobs"),
		client:  transcribe.NewFromConfig(cfg),
	}, nil
}

// List returns all Transcribe jobs.
func (d *JobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	jobs, err := appaws.Paginate(ctx, func(token *string) ([]types.TranscriptionJobSummary, *string, error) {
		output, err := d.client.ListTranscriptionJobs(ctx, &transcribe.ListTranscriptionJobsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list transcription jobs")
		}
		return output.TranscriptionJobSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(jobs))
	for i, job := range jobs {
		resources[i] = NewJobResource(job)
	}
	return resources, nil
}

// Get returns a specific Transcribe job by name.
func (d *JobDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetTranscriptionJob(ctx, &transcribe.GetTranscriptionJobInput{
		TranscriptionJobName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get transcription job %s", id)
	}
	return NewJobResourceFromDetail(*output.TranscriptionJob), nil
}

// Delete deletes a Transcribe job by name.
func (d *JobDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteTranscriptionJob(ctx, &transcribe.DeleteTranscriptionJobInput{
		TranscriptionJobName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete transcription job %s", id)
	}
	return nil
}

// JobResource wraps a Transcribe job.
type JobResource struct {
	dao.BaseResource
	Summary *types.TranscriptionJobSummary
	Detail  *types.TranscriptionJob
}

// NewJobResource creates a new JobResource from summary.
func NewJobResource(job types.TranscriptionJobSummary) *JobResource {
	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(job.TranscriptionJobName),
			ARN: "",
		},
		Summary: &job,
	}
}

// NewJobResourceFromDetail creates a new JobResource from detail.
func NewJobResourceFromDetail(job types.TranscriptionJob) *JobResource {
	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(job.TranscriptionJobName),
			ARN: "",
		},
		Detail: &job,
	}
}

// JobName returns the job name.
func (r *JobResource) JobName() string {
	return r.ID
}

// Status returns the job status.
func (r *JobResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.TranscriptionJobStatus)
	}
	if r.Detail != nil {
		return string(r.Detail.TranscriptionJobStatus)
	}
	return ""
}

// LanguageCode returns the language code.
func (r *JobResource) LanguageCode() string {
	if r.Summary != nil {
		return string(r.Summary.LanguageCode)
	}
	if r.Detail != nil {
		return string(r.Detail.LanguageCode)
	}
	return ""
}

// OutputLocationType returns the output location type.
func (r *JobResource) OutputLocationType() string {
	if r.Summary != nil {
		return string(r.Summary.OutputLocationType)
	}
	return ""
}

// CreationTime returns when the job was created.
func (r *JobResource) CreationTime() *time.Time {
	if r.Summary != nil {
		return r.Summary.CreationTime
	}
	if r.Detail != nil {
		return r.Detail.CreationTime
	}
	return nil
}

// CompletionTime returns when the job completed.
func (r *JobResource) CompletionTime() *time.Time {
	if r.Summary != nil {
		return r.Summary.CompletionTime
	}
	if r.Detail != nil {
		return r.Detail.CompletionTime
	}
	return nil
}

// FailureReason returns the failure reason.
func (r *JobResource) FailureReason() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.FailureReason)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.FailureReason)
	}
	return ""
}

// MediaFormat returns the media format.
func (r *JobResource) MediaFormat() string {
	if r.Detail != nil {
		return string(r.Detail.MediaFormat)
	}
	return ""
}

// MediaSampleRateHertz returns the media sample rate.
func (r *JobResource) MediaSampleRateHertz() int32 {
	if r.Detail != nil {
		return appaws.Int32(r.Detail.MediaSampleRateHertz)
	}
	return 0
}

// MediaFileUri returns the media file URI.
func (r *JobResource) MediaFileUri() string {
	if r.Detail != nil && r.Detail.Media != nil {
		return appaws.Str(r.Detail.Media.MediaFileUri)
	}
	return ""
}

// TranscriptFileUri returns the transcript file URI.
func (r *JobResource) TranscriptFileUri() string {
	if r.Detail != nil && r.Detail.Transcript != nil {
		return appaws.Str(r.Detail.Transcript.TranscriptFileUri)
	}
	return ""
}

// StartTime returns when the job started processing.
func (r *JobResource) StartTime() *time.Time {
	if r.Summary != nil {
		return r.Summary.StartTime
	}
	if r.Detail != nil {
		return r.Detail.StartTime
	}
	return nil
}

// IdentifiedLanguageScore returns the confidence score for auto language detection.
func (r *JobResource) IdentifiedLanguageScore() float32 {
	if r.Detail != nil && r.Detail.IdentifiedLanguageScore != nil {
		return *r.Detail.IdentifiedLanguageScore
	}
	return 0
}

// IdentifyLanguage returns whether language identification is enabled.
func (r *JobResource) IdentifyLanguage() bool {
	if r.Summary != nil {
		return appaws.Bool(r.Summary.IdentifyLanguage)
	}
	if r.Detail != nil {
		return appaws.Bool(r.Detail.IdentifyLanguage)
	}
	return false
}

// IdentifyMultipleLanguages returns whether multiple language identification is enabled.
func (r *JobResource) IdentifyMultipleLanguages() bool {
	if r.Summary != nil {
		return appaws.Bool(r.Summary.IdentifyMultipleLanguages)
	}
	if r.Detail != nil {
		return appaws.Bool(r.Detail.IdentifyMultipleLanguages)
	}
	return false
}

// ContentRedaction returns whether content redaction is enabled.
func (r *JobResource) ContentRedactionEnabled() bool {
	if r.Detail != nil && r.Detail.ContentRedaction != nil {
		return true
	}
	return false
}

// Subtitles returns whether subtitles are enabled.
func (r *JobResource) SubtitlesEnabled() bool {
	if r.Detail != nil && r.Detail.Subtitles != nil {
		return len(r.Detail.Subtitles.Formats) > 0
	}
	return false
}

// SubtitleFormats returns the subtitle formats.
func (r *JobResource) SubtitleFormats() []string {
	if r.Detail != nil && r.Detail.Subtitles != nil {
		formats := make([]string, len(r.Detail.Subtitles.Formats))
		for i, f := range r.Detail.Subtitles.Formats {
			formats[i] = string(f)
		}
		return formats
	}
	return nil
}

// ModelSettings returns the custom model name.
func (r *JobResource) ModelName() string {
	if r.Detail != nil && r.Detail.ModelSettings != nil {
		return appaws.Str(r.Detail.ModelSettings.LanguageModelName)
	}
	return ""
}
