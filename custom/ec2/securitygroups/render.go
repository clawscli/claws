package securitygroups

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure SecurityGroupRenderer implements render.Navigator
var _ render.Navigator = (*SecurityGroupRenderer)(nil)

// SecurityGroupRenderer renders EC2 security groups
type SecurityGroupRenderer struct {
	render.BaseRenderer
}

// NewSecurityGroupRenderer creates a new SecurityGroupRenderer
func NewSecurityGroupRenderer() render.Renderer {
	return &SecurityGroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "security-groups",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 25,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "VPC",
					Width: 22,
					Getter: func(r dao.Resource) string {
						if sg, ok := r.(*SecurityGroupResource); ok {
							return sg.VpcID()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "INBOUND",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if sg, ok := r.(*SecurityGroupResource); ok {
							return fmt.Sprintf("%d", sg.InboundRuleCount())
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "OUTBOUND",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if sg, ok := r.(*SecurityGroupResource); ok {
							return fmt.Sprintf("%d", sg.OutboundRuleCount())
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "DESCRIPTION",
					Width: 40,
					Getter: func(r dao.Resource) string {
						if sg, ok := r.(*SecurityGroupResource); ok {
							desc := sg.Description()
							if len(desc) > 40 {
								desc = desc[:37] + "..."
							}
							return desc
						}
						return ""
					},
					Priority: 5,
				},
				render.TagsColumn(30, 6),
			},
		},
	}
}

// RenderDetail renders detailed security group information
func (r *SecurityGroupRenderer) RenderDetail(resource dao.Resource) string {
	sg, ok := resource.(*SecurityGroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Security Group", sg.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Group ID", sg.GetID())
	d.Field("Group Name", sg.GetName())
	d.Field("VPC ID", sg.VpcID())
	if sg.Item.OwnerId != nil {
		d.Field("Owner ID", *sg.Item.OwnerId)
	}
	d.Field("Description", sg.Description())

	// Inbound rules
	d.Section(fmt.Sprintf("Inbound Rules (%d)", len(sg.Item.IpPermissions)))
	if len(sg.Item.IpPermissions) > 0 {
		for _, perm := range sg.Item.IpPermissions {
			d.Line(formatPermission(perm))
		}
	} else {
		d.DimIndent("(none)")
	}

	// Outbound rules
	d.Section(fmt.Sprintf("Outbound Rules (%d)", len(sg.Item.IpPermissionsEgress)))
	if len(sg.Item.IpPermissionsEgress) > 0 {
		for _, perm := range sg.Item.IpPermissionsEgress {
			d.Line(formatPermission(perm))
		}
	} else {
		d.DimIndent("(none)")
	}

	// Tags
	d.Tags(sg.GetTags())

	return d.String()
}

func formatPermission(perm types.IpPermission) string {
	var parts []string

	// Protocol
	proto := appaws.Str(perm.IpProtocol)
	if proto == "" || proto == "-1" {
		proto = "All"
	}

	// Port range
	portRange := "All"
	if perm.FromPort != nil && perm.ToPort != nil {
		from := *perm.FromPort
		to := *perm.ToPort
		if from == to {
			if from == -1 {
				portRange = "All"
			} else {
				portRange = fmt.Sprintf("%d", from)
			}
		} else {
			portRange = fmt.Sprintf("%d-%d", from, to)
		}
	}

	parts = append(parts, fmt.Sprintf("%-6s  Port: %-11s", proto, portRange))

	// Source/Destination
	var sources []string

	// CIDR ranges
	for _, ipRange := range perm.IpRanges {
		src := appaws.Str(ipRange.CidrIp)
		if src != "" {
			if desc := appaws.Str(ipRange.Description); desc != "" {
				src += " (" + desc + ")"
			}
			sources = append(sources, src)
		}
	}

	// IPv6 CIDR ranges
	for _, ipRange := range perm.Ipv6Ranges {
		src := appaws.Str(ipRange.CidrIpv6)
		if src != "" {
			if desc := appaws.Str(ipRange.Description); desc != "" {
				src += " (" + desc + ")"
			}
			sources = append(sources, src)
		}
	}

	// Security group references
	for _, sg := range perm.UserIdGroupPairs {
		src := appaws.Str(sg.GroupId)
		if src != "" {
			if desc := appaws.Str(sg.Description); desc != "" {
				src += " (" + desc + ")"
			}
			sources = append(sources, src)
		}
	}

	// Prefix lists
	for _, pl := range perm.PrefixListIds {
		src := appaws.Str(pl.PrefixListId)
		if src != "" {
			if desc := appaws.Str(pl.Description); desc != "" {
				src += " (" + desc + ")"
			}
			sources = append(sources, src)
		}
	}

	if len(sources) > 0 {
		parts = append(parts, strings.Join(sources, ", "))
	}

	return "  " + strings.Join(parts, "  ")
}

// RenderSummary returns summary fields for the header panel
func (r *SecurityGroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sg, ok := resource.(*SecurityGroupResource)
	if !ok {
		return nil
	}

	// Row 1: ID, Name, VPC
	fields := []render.SummaryField{
		{Label: "ID", Value: sg.GetID()},
		{Label: "Name", Value: sg.GetName()},
		{Label: "VPC", Value: sg.VpcID()},
	}

	// Row 2: Rule counts and description
	fields = append(fields,
		render.SummaryField{Label: "Inbound", Value: fmt.Sprintf("%d rules", sg.InboundRuleCount())},
		render.SummaryField{Label: "Outbound", Value: fmt.Sprintf("%d rules", sg.OutboundRuleCount())},
	)

	// Row 3: Description (truncated if too long)
	if desc := sg.Description(); desc != "" {
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	// Row 4: Owner ID if available
	if sg.Item.OwnerId != nil {
		fields = append(fields, render.SummaryField{Label: "Owner", Value: *sg.Item.OwnerId})
	}

	return fields
}

// Navigations returns navigation shortcuts for Security Group resources
func (r *SecurityGroupRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sg, ok := resource.(*SecurityGroupResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// VPC navigation
	if sg.Item.VpcId != nil {
		navs = append(navs, render.Navigation{
			Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs",
			FilterField: "VpcId", FilterValue: *sg.Item.VpcId,
		})
		// Instances in same VPC
		navs = append(navs, render.Navigation{
			Key: "e", Label: "Instances", Service: "ec2", Resource: "instances",
			FilterField: "VpcId", FilterValue: *sg.Item.VpcId,
		})
	}

	return navs
}
