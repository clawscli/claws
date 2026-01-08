package taskdefinitions

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

var _ render.Navigator = (*TaskDefinitionRenderer)(nil)

type TaskDefinitionRenderer struct {
	render.BaseRenderer
}

func NewTaskDefinitionRenderer() render.Renderer {
	return &TaskDefinitionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ecs",
			Resource: "task-definitions",
			Cols: []render.Column{
				{Name: "FAMILY", Width: 35, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "REV", Width: 5, Getter: getRevision},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "CPU", Width: 8, Getter: getCPU},
				{Name: "MEMORY", Width: 8, Getter: getMemory},
				{Name: "NETWORK", Width: 10, Getter: getNetworkMode},
				{Name: "CONTAINERS", Width: 10, Getter: getContainerCount},
			},
		},
	}
}

func getRevision(r dao.Resource) string {
	if td, ok := r.(*TaskDefinitionResource); ok {
		return fmt.Sprintf("%d", td.Revision())
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if td, ok := r.(*TaskDefinitionResource); ok {
		status := td.Status()
		switch status {
		case "ACTIVE":
			return "active"
		case "INACTIVE":
			return "stopped"
		case "DELETE_IN_PROGRESS":
			return "deleting"
		default:
			return strings.ToLower(status)
		}
	}
	return ""
}

func getCPU(r dao.Resource) string {
	if td, ok := r.(*TaskDefinitionResource); ok {
		if cpu := td.CPU(); cpu != "" {
			return cpu
		}
	}
	return "-"
}

func getMemory(r dao.Resource) string {
	if td, ok := r.(*TaskDefinitionResource); ok {
		if mem := td.Memory(); mem != "" {
			return mem
		}
	}
	return "-"
}

func getNetworkMode(r dao.Resource) string {
	if td, ok := r.(*TaskDefinitionResource); ok {
		mode := td.NetworkMode()
		if mode == "" {
			return "bridge"
		}
		return mode
	}
	return ""
}

func getContainerCount(r dao.Resource) string {
	if td, ok := r.(*TaskDefinitionResource); ok {
		return fmt.Sprintf("%d", len(td.ContainerDefinitions()))
	}
	return ""
}

func (r *TaskDefinitionRenderer) RenderDetail(resource dao.Resource) string {
	td, ok := resource.(*TaskDefinitionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ECS Task Definition", td.GetID())

	d.Section("Basic Information")
	d.Field("Family", td.Family())
	d.Field("Revision", fmt.Sprintf("%d", td.Revision()))
	d.Field("ARN", td.GetARN())
	d.FieldStyled("Status", td.Status(), render.StateColorer()(strings.ToLower(td.Status())))

	d.Section("Task Configuration")
	if cpu := td.CPU(); cpu != "" {
		d.Field("CPU", cpu+" units")
	}
	if mem := td.Memory(); mem != "" {
		d.Field("Memory", mem+" MiB")
	}
	d.Field("Network Mode", td.NetworkMode())

	if compat := td.RequiresCompatibilities(); len(compat) > 0 {
		var compatStr []string
		for _, c := range compat {
			compatStr = append(compatStr, string(c))
		}
		d.Field("Compatibilities", strings.Join(compatStr, ", "))
	}

	if rp := td.RuntimePlatform(); rp != nil {
		if rp.OperatingSystemFamily != "" {
			d.Field("OS Family", string(rp.OperatingSystemFamily))
		}
		if rp.CpuArchitecture != "" {
			d.Field("CPU Architecture", string(rp.CpuArchitecture))
		}
	}

	if role := td.TaskRoleArn(); role != "" {
		d.Section("IAM Roles")
		d.Field("Task Role", appaws.ExtractResourceName(role))
	}
	if execRole := td.ExecutionRoleArn(); execRole != "" {
		if td.TaskRoleArn() == "" {
			d.Section("IAM Roles")
		}
		d.Field("Execution Role", appaws.ExtractResourceName(execRole))
	}

	containers := td.ContainerDefinitions()
	if len(containers) > 0 {
		d.Section(fmt.Sprintf("Containers (%d)", len(containers)))
		for _, c := range containers {
			containerName := appaws.Str(c.Name)
			d.Line("")
			d.FieldStyled(containerName, "", ui.TitleStyle())

			if c.Image != nil {
				d.Field("  Image", *c.Image)
			}
			if c.Essential != nil && *c.Essential {
				d.Field("  Essential", "Yes")
			}
			if c.Cpu != 0 {
				d.Field("  CPU", fmt.Sprintf("%d", c.Cpu))
			}
			if c.Memory != nil {
				d.Field("  Memory", fmt.Sprintf("%d MiB", *c.Memory))
			}
			if c.MemoryReservation != nil {
				d.Field("  Memory Reservation", fmt.Sprintf("%d MiB", *c.MemoryReservation))
			}

			if len(c.PortMappings) > 0 {
				var ports []string
				for _, pm := range c.PortMappings {
					if pm.ContainerPort != nil {
						port := fmt.Sprintf("%d", *pm.ContainerPort)
						if pm.HostPort != nil && *pm.HostPort != *pm.ContainerPort {
							port = fmt.Sprintf("%d:%d", *pm.HostPort, *pm.ContainerPort)
						}
						if pm.Protocol != "" {
							port += "/" + strings.ToLower(string(pm.Protocol))
						}
						ports = append(ports, port)
					}
				}
				d.Field("  Ports", strings.Join(ports, ", "))
			}

			if c.LogConfiguration != nil {
				d.Field("  Log Driver", string(c.LogConfiguration.LogDriver))
				if c.LogConfiguration.LogDriver == types.LogDriverAwslogs && c.LogConfiguration.Options != nil {
					if group := c.LogConfiguration.Options["awslogs-group"]; group != "" {
						d.FieldStyled("  Log Group", group, ui.SuccessStyle())
					}
					if prefix := c.LogConfiguration.Options["awslogs-stream-prefix"]; prefix != "" {
						d.Field("  Stream Prefix", prefix)
					}
				}
			}
		}
	}

	if volumes := td.Volumes(); len(volumes) > 0 {
		d.Section("Volumes")
		for _, v := range volumes {
			if v.Name != nil {
				d.Field(*v.Name, "")
				if v.Host != nil && v.Host.SourcePath != nil {
					d.Field("  Source", *v.Host.SourcePath)
				}
				if v.EfsVolumeConfiguration != nil {
					d.Field("  Type", "EFS")
					d.Field("  File System ID", appaws.Str(v.EfsVolumeConfiguration.FileSystemId))
				}
			}
		}
	}

	d.Tags(td.GetTags())

	return d.String()
}

func (r *TaskDefinitionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	td, ok := resource.(*TaskDefinitionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Family", Value: td.Family()},
		{Label: "Revision", Value: fmt.Sprintf("%d", td.Revision())},
		{Label: "ARN", Value: td.GetARN()},
		{Label: "Status", Value: td.Status()},
	}

	if cpu := td.CPU(); cpu != "" {
		fields = append(fields, render.SummaryField{Label: "CPU", Value: cpu})
	}
	if mem := td.Memory(); mem != "" {
		fields = append(fields, render.SummaryField{Label: "Memory", Value: mem})
	}

	fields = append(fields, render.SummaryField{Label: "Network Mode", Value: td.NetworkMode()})
	fields = append(fields, render.SummaryField{Label: "Containers", Value: fmt.Sprintf("%d", len(td.ContainerDefinitions()))})

	if groups := td.GetAllCloudWatchLogGroups(); len(groups) > 0 {
		fields = append(fields, render.SummaryField{Label: "Log Groups", Value: strings.Join(groups, ", ")})
	}

	return fields
}

func (r *TaskDefinitionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	td, ok := resource.(*TaskDefinitionResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	if groups := td.GetAllCloudWatchLogGroups(); len(groups) > 0 {
		navs = append(navs, render.Navigation{
			Key:         "l",
			Label:       "Logs",
			Service:     "cloudwatch",
			Resource:    "log-groups",
			FilterField: "LogGroupPrefix",
			FilterValue: groups[0],
		})
	}

	if role := td.TaskRoleArn(); role != "" {
		navs = append(navs, render.Navigation{
			Key:         "r",
			Label:       "Task Role",
			Service:     "iam",
			Resource:    "roles",
			FilterField: "RoleName",
			FilterValue: appaws.ExtractResourceName(role),
		})
	}

	return navs
}
