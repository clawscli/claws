package trainingjobs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// TrainingJobDAO provides data access for SageMaker training jobs.
type TrainingJobDAO struct {
	dao.BaseDAO
	client *sagemaker.Client
}

// NewTrainingJobDAO creates a new TrainingJobDAO.
func NewTrainingJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new sagemaker/trainingjobs dao")
	}
	return &TrainingJobDAO{
		BaseDAO: dao.NewBaseDAO("sagemaker", "training-jobs"),
		client:  sagemaker.NewFromConfig(cfg),
	}, nil
}

// List returns all SageMaker training jobs.
func (d *TrainingJobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	jobs, err := appaws.Paginate(ctx, func(token *string) ([]types.TrainingJobSummary, *string, error) {
		output, err := d.client.ListTrainingJobs(ctx, &sagemaker.ListTrainingJobsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list sagemaker training jobs")
		}
		return output.TrainingJobSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(jobs))
	for i, job := range jobs {
		resources[i] = NewTrainingJobResource(job)
	}
	return resources, nil
}

// Get returns a specific training job.
func (d *TrainingJobDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeTrainingJob(ctx, &sagemaker.DescribeTrainingJobInput{
		TrainingJobName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe sagemaker training job")
	}
	// Convert to summary for consistent resource type
	summary := types.TrainingJobSummary{
		TrainingJobName:   output.TrainingJobName,
		TrainingJobArn:    output.TrainingJobArn,
		TrainingJobStatus: output.TrainingJobStatus,
		CreationTime:      output.CreationTime,
		TrainingEndTime:   output.TrainingEndTime,
	}
	r := NewTrainingJobResource(summary)
	r.FailureReason = appaws.Str(output.FailureReason)
	r.RoleArn = appaws.Str(output.RoleArn)
	r.TrainingStartTime = output.TrainingStartTime
	r.LastModifiedTime = output.LastModifiedTime
	r.SecondaryStatus = string(output.SecondaryStatus)
	if output.BillableTimeInSeconds != nil {
		r.BillableTimeInSeconds = *output.BillableTimeInSeconds
	}
	if output.TrainingTimeInSeconds != nil {
		r.TrainingTimeInSeconds = *output.TrainingTimeInSeconds
	}
	if output.ModelArtifacts != nil {
		r.ModelArtifactsS3 = appaws.Str(output.ModelArtifacts.S3ModelArtifacts)
	}
	if output.AlgorithmSpecification != nil {
		r.AlgorithmTrainingImage = appaws.Str(output.AlgorithmSpecification.TrainingImage)
		if output.AlgorithmSpecification.AlgorithmName != nil {
			r.AlgorithmImage = *output.AlgorithmSpecification.AlgorithmName
		}
	}
	if output.ResourceConfig != nil {
		r.InstanceType = string(output.ResourceConfig.InstanceType)
		if output.ResourceConfig.InstanceCount != nil {
			r.InstanceCount = *output.ResourceConfig.InstanceCount
		}
		if output.ResourceConfig.VolumeSizeInGB != nil {
			r.VolumeSizeInGB = *output.ResourceConfig.VolumeSizeInGB
		}
	}
	if output.StoppingCondition != nil && output.StoppingCondition.MaxRuntimeInSeconds != nil {
		r.MaxRuntimeSeconds = *output.StoppingCondition.MaxRuntimeInSeconds
	}
	r.HyperParameters = output.HyperParameters
	r.InputDataConfig = output.InputDataConfig
	if output.OutputDataConfig != nil {
		r.OutputS3Path = appaws.Str(output.OutputDataConfig.S3OutputPath)
	}
	if output.EnableManagedSpotTraining != nil {
		r.EnableSpotTraining = *output.EnableManagedSpotTraining
	}
	if output.EnableNetworkIsolation != nil {
		r.EnableNetworkIsolation = *output.EnableNetworkIsolation
	}
	r.TuningJobArn = appaws.Str(output.TuningJobArn)
	r.FinalMetrics = output.FinalMetricDataList
	return r, nil
}

// Delete stops a training job (training jobs can't be deleted, only stopped).
func (d *TrainingJobDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.StopTrainingJob(ctx, &sagemaker.StopTrainingJobInput{
		TrainingJobName: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "stop sagemaker training job")
	}
	return nil
}

// TrainingJobResource wraps a SageMaker training job.
type TrainingJobResource struct {
	dao.BaseResource
	Job                    types.TrainingJobSummary
	FailureReason          string
	RoleArn                string
	TrainingStartTime      *time.Time
	LastModifiedTime       *time.Time
	SecondaryStatus        string
	BillableTimeInSeconds  int32
	TrainingTimeInSeconds  int32
	ModelArtifactsS3       string
	AlgorithmImage         string
	AlgorithmTrainingImage string
	InstanceType           string
	InstanceCount          int32
	VolumeSizeInGB         int32
	MaxRuntimeSeconds      int32
	HyperParameters        map[string]string
	InputDataConfig        []types.Channel
	OutputS3Path           string
	EnableSpotTraining     bool
	EnableNetworkIsolation bool
	TuningJobArn           string
	FinalMetrics           []types.MetricData
}

// NewTrainingJobResource creates a new TrainingJobResource.
func NewTrainingJobResource(job types.TrainingJobSummary) *TrainingJobResource {
	return &TrainingJobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(job.TrainingJobName),
			ARN: appaws.Str(job.TrainingJobArn),
		},
		Job: job,
	}
}

// Status returns the training job status.
func (r *TrainingJobResource) Status() string {
	return string(r.Job.TrainingJobStatus)
}

// CreatedAt returns when the training job was created.
func (r *TrainingJobResource) CreatedAt() *time.Time {
	return r.Job.CreationTime
}

// TrainingEndTime returns when the training job ended.
func (r *TrainingJobResource) TrainingEndTime() *time.Time {
	return r.Job.TrainingEndTime
}

// GetFailureReason returns the failure reason.
func (r *TrainingJobResource) GetFailureReason() string {
	return r.FailureReason
}

// GetRoleArn returns the IAM role ARN.
func (r *TrainingJobResource) GetRoleArn() string {
	return r.RoleArn
}

// GetTrainingStartTime returns when training started.
func (r *TrainingJobResource) GetTrainingStartTime() *time.Time {
	return r.TrainingStartTime
}

// GetLastModifiedTime returns when the job was last modified.
func (r *TrainingJobResource) GetLastModifiedTime() *time.Time {
	return r.LastModifiedTime
}

// GetSecondaryStatus returns the secondary status.
func (r *TrainingJobResource) GetSecondaryStatus() string {
	return r.SecondaryStatus
}

// GetBillableTimeInSeconds returns billable time.
func (r *TrainingJobResource) GetBillableTimeInSeconds() int32 {
	return r.BillableTimeInSeconds
}

// GetTrainingTimeInSeconds returns training time.
func (r *TrainingJobResource) GetTrainingTimeInSeconds() int32 {
	return r.TrainingTimeInSeconds
}

// GetModelArtifactsS3 returns the model artifacts S3 location.
func (r *TrainingJobResource) GetModelArtifactsS3() string {
	return r.ModelArtifactsS3
}

// GetAlgorithmImage returns the algorithm image.
func (r *TrainingJobResource) GetAlgorithmImage() string {
	return r.AlgorithmImage
}

// GetAlgorithmTrainingImage returns the training image.
func (r *TrainingJobResource) GetAlgorithmTrainingImage() string {
	return r.AlgorithmTrainingImage
}

// GetInstanceType returns the instance type.
func (r *TrainingJobResource) GetInstanceType() string {
	return r.InstanceType
}

// GetInstanceCount returns the instance count.
func (r *TrainingJobResource) GetInstanceCount() int32 {
	return r.InstanceCount
}

// GetVolumeSizeInGB returns the volume size.
func (r *TrainingJobResource) GetVolumeSizeInGB() int32 {
	return r.VolumeSizeInGB
}

// GetMaxRuntimeSeconds returns the max runtime in seconds.
func (r *TrainingJobResource) GetMaxRuntimeSeconds() int32 {
	return r.MaxRuntimeSeconds
}

// GetHyperParameters returns the hyperparameters.
func (r *TrainingJobResource) GetHyperParameters() map[string]string {
	return r.HyperParameters
}

// GetInputDataConfig returns input data channels.
func (r *TrainingJobResource) GetInputDataConfig() []types.Channel {
	return r.InputDataConfig
}

// GetOutputS3Path returns the output S3 path.
func (r *TrainingJobResource) GetOutputS3Path() string {
	return r.OutputS3Path
}

// GetEnableSpotTraining returns whether spot training is enabled.
func (r *TrainingJobResource) GetEnableSpotTraining() bool {
	return r.EnableSpotTraining
}

// GetEnableNetworkIsolation returns whether network isolation is enabled.
func (r *TrainingJobResource) GetEnableNetworkIsolation() bool {
	return r.EnableNetworkIsolation
}

// GetTuningJobArn returns the tuning job ARN if any.
func (r *TrainingJobResource) GetTuningJobArn() string {
	return r.TuningJobArn
}

// GetFinalMetrics returns the final training metrics.
func (r *TrainingJobResource) GetFinalMetrics() []types.MetricData {
	return r.FinalMetrics
}
