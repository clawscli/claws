package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ActivityDAO provides data access for Auto Scaling activities
type ActivityDAO struct {
	dao.BaseDAO
	client *autoscaling.Client
}

// NewActivityDAO creates a new ActivityDAO
func NewActivityDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new autoscaling/activities dao: %w", err)
	}
	return &ActivityDAO{
		BaseDAO: dao.NewBaseDAO("autoscaling", "activities"),
		client:  autoscaling.NewFromConfig(cfg),
	}, nil
}

// List returns activities (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *ActivityDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of Auto Scaling activities.
// Implements dao.PaginatedDAO interface.
func (d *ActivityDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get ASG name from filter context
	asgName := dao.GetFilterFromContext(ctx, "AutoScalingGroupName")
	if asgName == "" {
		return nil, "", fmt.Errorf("auto scaling group name filter required")
	}

	maxRecords := int32(pageSize)
	if maxRecords > 100 {
		maxRecords = 100 // AWS API max
	}

	input := &autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: &asgName,
		MaxRecords:           &maxRecords,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeScalingActivities(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("describe scaling activities: %w", err)
	}

	resources := make([]dao.Resource, len(output.Activities))
	for i, activity := range output.Activities {
		resources[i] = NewActivityResource(activity, asgName)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific activity
func (d *ActivityDAO) Get(ctx context.Context, activityId string) (dao.Resource, error) {
	asgName := dao.GetFilterFromContext(ctx, "AutoScalingGroupName")

	input := &autoscaling.DescribeScalingActivitiesInput{
		ActivityIds: []string{activityId},
	}

	output, err := d.client.DescribeScalingActivities(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe activity %s: %w", activityId, err)
	}

	if len(output.Activities) == 0 {
		return nil, fmt.Errorf("activity not found: %s", activityId)
	}

	return NewActivityResource(output.Activities[0], asgName), nil
}

// Delete is not supported for activities
func (d *ActivityDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for scaling activities")
}

// Supports returns supported operations
func (d *ActivityDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}

// ActivityResource represents an Auto Scaling activity
type ActivityResource struct {
	dao.BaseResource
	Activity             types.Activity
	AutoScalingGroupName string
}

// NewActivityResource creates a new ActivityResource
func NewActivityResource(activity types.Activity, asgName string) *ActivityResource {
	id := appaws.Str(activity.ActivityId)

	return &ActivityResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: activity,
		},
		Activity:             activity,
		AutoScalingGroupName: asgName,
	}
}

// ActivityId returns the activity ID
func (r *ActivityResource) ActivityId() string {
	return appaws.Str(r.Activity.ActivityId)
}

// StatusCode returns the status code
func (r *ActivityResource) StatusCode() string {
	return string(r.Activity.StatusCode)
}

// StatusMessage returns the status message
func (r *ActivityResource) StatusMessage() string {
	return appaws.Str(r.Activity.StatusMessage)
}

// Cause returns the cause of the activity
func (r *ActivityResource) Cause() string {
	return appaws.Str(r.Activity.Cause)
}

// Description returns the description
func (r *ActivityResource) Description() string {
	return appaws.Str(r.Activity.Description)
}

// Progress returns the progress percentage
func (r *ActivityResource) Progress() int32 {
	if r.Activity.Progress != nil {
		return *r.Activity.Progress
	}
	return 0
}

// StartTime returns the start time
func (r *ActivityResource) StartTime() string {
	if r.Activity.StartTime != nil {
		return r.Activity.StartTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// StartTimeT returns the start time as time.Time
func (r *ActivityResource) StartTimeT() *time.Time {
	return r.Activity.StartTime
}

// EndTime returns the end time
func (r *ActivityResource) EndTime() string {
	if r.Activity.EndTime != nil {
		return r.Activity.EndTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// Duration returns the duration of the activity
func (r *ActivityResource) Duration() string {
	if r.Activity.StartTime == nil {
		return ""
	}
	var endTime time.Time
	if r.Activity.EndTime != nil {
		endTime = *r.Activity.EndTime
	} else {
		endTime = time.Now()
	}
	dur := endTime.Sub(*r.Activity.StartTime)
	if dur < time.Minute {
		return fmt.Sprintf("%ds", int(dur.Seconds()))
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm%ds", int(dur.Minutes()), int(dur.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(dur.Hours()), int(dur.Minutes())%60)
}

// Details returns activity details
func (r *ActivityResource) Details() string {
	return appaws.Str(r.Activity.Details)
}

// ASGName returns the Auto Scaling Group name from the activity
func (r *ActivityResource) ASGName() string {
	return appaws.Str(r.Activity.AutoScalingGroupName)
}
