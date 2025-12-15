package resources

import (
	"strings"
	"time"
	"unicode"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

// Ensure ResourceRenderer implements render.Navigator
var _ render.Navigator = (*ResourceRenderer)(nil)

// ResourceRenderer renders CloudFormation stack resources
type ResourceRenderer struct {
	render.BaseRenderer
}

// NewResourceRenderer creates a new ResourceRenderer
func NewResourceRenderer() render.Renderer {
	return &ResourceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudformation",
			Resource: "resources",
			Cols: []render.Column{
				{
					Name:  "LOGICAL ID",
					Width: 30,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "TYPE",
					Width: 35,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StackResourceResource); ok {
							return rr.ResourceType()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "STATUS",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StackResourceResource); ok {
							return rr.ResourceStatus()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "PHYSICAL ID",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 3,
				},
			},
		},
	}
}

// RenderDetail renders detailed resource information
func (r *ResourceRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*StackResourceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Stack Resource", rr.GetName())

	d.Section("Resource Information")
	d.Field("Logical Resource ID", rr.GetName())
	d.Field("Physical Resource ID", rr.GetID())
	d.Field("Resource Type", rr.ResourceType())
	d.FieldStyled("Status", rr.ResourceStatus(), cfnResourceStatusColorer(rr.ResourceStatus()))

	if rr.Item.Timestamp != nil {
		d.Field("Last Updated", rr.Item.Timestamp.Format(time.RFC3339))
	}

	if rr.StatusReason() != "" {
		d.Section("Status Reason")
		d.Line("  " + rr.StatusReason())
	}

	if rr.DriftStatus() != "" {
		d.Section("Drift Information")
		d.FieldStyled("Drift Status", rr.DriftStatus(), driftColorer(rr.DriftStatus()))
	}

	d.FieldIf("Stack Name", rr.Item.StackName)
	d.FieldIf("Description", rr.Item.Description)

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ResourceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*StackResourceResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Logical ID", Value: rr.GetName()},
		{Label: "Type", Value: rr.ResourceType()},
		{Label: "Status", Value: rr.ResourceStatus(), Style: cfnResourceStatusColorer(rr.ResourceStatus())},
	}

	if rr.GetID() != "" {
		physicalId := rr.GetID()
		if len(physicalId) > 50 {
			physicalId = physicalId[:47] + "..."
		}
		fields = append(fields, render.SummaryField{Label: "Physical ID", Value: physicalId})
	}

	if rr.DriftStatus() != "" {
		fields = append(fields, render.SummaryField{
			Label: "Drift",
			Value: rr.DriftStatus(),
			Style: driftColorer(rr.DriftStatus()),
		})
	}

	if rr.Item.Timestamp != nil {
		fields = append(fields, render.SummaryField{
			Label: "Updated",
			Value: rr.Item.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}

	return fields
}

// cfnResourceStatusColorer returns a style for CloudFormation resource status
func cfnResourceStatusColorer(status string) render.Style {
	switch {
	case strings.HasSuffix(status, "_COMPLETE") && !strings.Contains(status, "ROLLBACK") && !strings.Contains(status, "DELETE"):
		return render.SuccessStyle()
	case strings.Contains(status, "IN_PROGRESS"):
		return render.WarningStyle()
	case strings.Contains(status, "FAILED") || strings.Contains(status, "ROLLBACK"):
		return render.DangerStyle()
	case strings.Contains(status, "DELETE_COMPLETE") || strings.Contains(status, "SKIPPED"):
		return render.DimStyle()
	default:
		return render.DefaultStyle()
	}
}

// driftColorer returns a style for drift status
func driftColorer(status string) render.Style {
	switch status {
	case "IN_SYNC":
		return render.SuccessStyle()
	case "MODIFIED", "DELETED":
		return render.DangerStyle()
	case "NOT_CHECKED":
		return render.DimStyle()
	default:
		return render.DefaultStyle()
	}
}

// Navigations returns navigation shortcuts for CloudFormation resources
// It automatically maps AWS resource types to claws services if they exist
func (r *ResourceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*StackResourceResource)
	if !ok {
		return nil
	}

	// Get the physical resource ID
	physicalID := rr.GetID()
	if physicalID == "" {
		return nil
	}

	// Parse the resource type (e.g., "AWS::EC2::Instance" → "ec2", "instances")
	service, resourceType := parseCfnResourceType(rr.ResourceType())
	if service == "" || resourceType == "" {
		return nil
	}

	// Check if the service/resource is registered in claws
	if !registry.Global.HasResource(service, resourceType) {
		return nil
	}

	// Determine the filter field based on resource type
	filterField := getFilterField(rr.ResourceType())

	// Extract the actual resource name from ARN if needed
	filterValue := extractFilterValue(physicalID, rr.ResourceType())

	return []render.Navigation{
		{
			Key:         "g",
			Label:       "Go to Resource",
			Service:     service,
			Resource:    resourceType,
			FilterField: filterField,
			FilterValue: filterValue,
		},
	}
}

// extractFilterValue extracts the appropriate filter value from the physical resource ID
// For ARN-based resources, this extracts the resource name from the ARN
func extractFilterValue(physicalID, cfnType string) string {
	// If it's an ARN, extract the resource name
	if strings.HasPrefix(physicalID, "arn:aws:") {
		// ARN format: arn:aws:service:region:account:resource-type/resource-name
		// or: arn:aws:service:region:account:resource-type:resource-name
		parts := strings.Split(physicalID, ":")
		if len(parts) >= 6 {
			resourcePart := strings.Join(parts[5:], ":")
			// Handle resource-type/resource-name format (e.g., role/MyRole)
			if idx := strings.LastIndex(resourcePart, "/"); idx != -1 {
				return resourcePart[idx+1:]
			}
			// Handle resource-type:resource-name format
			if idx := strings.LastIndex(resourcePart, ":"); idx != -1 {
				return resourcePart[idx+1:]
			}
			return resourcePart
		}
	}
	// For non-ARN resources (like S3 bucket names), use as-is
	return physicalID
}

// parseCfnResourceType converts a CloudFormation resource type to claws service/resource
// e.g., "AWS::EC2::Instance" → ("ec2", "instances")
func parseCfnResourceType(cfnType string) (service, resource string) {
	// Parse "AWS::Service::Resource"
	parts := strings.Split(cfnType, "::")
	if len(parts) != 3 || parts[0] != "AWS" {
		return "", ""
	}

	awsService := strings.ToLower(parts[1])
	awsResource := parts[2]

	// Map AWS service names to claws service names
	switch awsService {
	case "ec2":
		// Some EC2 resources are in the "vpc" service
		switch awsResource {
		case "VPC", "Subnet", "RouteTable", "InternetGateway", "NatGateway":
			service = "vpc"
		default:
			service = "ec2"
		}
	default:
		service = awsService
	}

	// Convert resource name: CamelCase → kebab-case, then pluralize
	resource = camelToKebab(awsResource)
	resource = pluralize(resource)

	return service, resource
}

// camelToKebab converts CamelCase to kebab-case
func camelToKebab(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('-')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// pluralize adds 's' or 'es' to make plural form
func pluralize(s string) string {
	// Handle special cases
	switch s {
	case "policy":
		return "policies"
	case "security-group":
		return "security-groups"
	case "instance-profile":
		return "instance-profiles"
	case "route-table":
		return "route-tables"
	case "internet-gateway":
		return "internet-gateways"
	case "nat-gateway":
		return "nat-gateways"
	}

	// Default: just add 's'
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

// getFilterField returns the appropriate filter field for the resource type
func getFilterField(cfnType string) string {
	parts := strings.Split(cfnType, "::")
	if len(parts) != 3 {
		return ""
	}

	awsResource := parts[2]

	// Map resource types to their ID field names
	switch awsResource {
	case "Instance":
		return "InstanceId"
	case "VPC":
		return "VpcId"
	case "Subnet":
		return "SubnetId"
	case "SecurityGroup":
		return "GroupId"
	case "Volume":
		return "VolumeId"
	case "RouteTable":
		return "RouteTableId"
	case "InternetGateway":
		return "InternetGatewayId"
	case "NatGateway":
		return "NatGatewayId"
	case "Role":
		return "RoleName"
	case "User":
		return "UserName"
	case "Policy":
		return "PolicyName"
	case "InstanceProfile":
		return "InstanceProfileName"
	case "Bucket":
		return "Name"
	default:
		// For most resources, use empty filter (show all and let user find)
		return ""
	}
}
