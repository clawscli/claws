package clusters

import (
	"encoding/json"
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

var _ render.Navigator = (*ClusterRenderer)(nil)

// formatBool converts bool to Yes/No string
func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// ClusterRenderer renders EKS cluster resources
type ClusterRenderer struct {
	render.BaseRenderer
}

// NewClusterRenderer creates a new ClusterRenderer
func NewClusterRenderer() render.Renderer {
	return &ClusterRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eks",
			Resource: "clusters",
			Cols: []render.Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 0,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
				},
				{
					Name:     "VERSION",
					Width:    10,
					Priority: 1,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*ClusterResource); ok {
							return cr.Version()
						}
						return ""
					},
				},
				{
					Name:     "STATUS",
					Width:    12,
					Priority: 2,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*ClusterResource); ok {
							return cr.Status()
						}
						return ""
					},
				},
				{
					Name:     "ENDPOINT",
					Width:    50,
					Priority: 3,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*ClusterResource); ok {
							return cr.Endpoint()
						}
						return ""
					},
				},
				{
					Name:     "AGE",
					Width:    10,
					Priority: 4,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*ClusterResource); ok {
							return cr.CreatedAge()
						}
						return ""
					},
				},
			},
		},
	}
}

func (rnd *ClusterRenderer) RenderDetail(resource dao.Resource) string {
	cr, ok := resource.(*ClusterResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("EKS Cluster", cr.GetName())

	// Basic Information
	d.Section("Basic Information")
	d.Field("Name", cr.GetName())
	d.Field("ARN", cr.GetARN())
	d.Field("Version", cr.Version())
	d.Field("Status", cr.Status())
	d.Field("Platform Version", cr.PlatformVersion())
	d.Field("Created", cr.CreatedAge())
	d.Field("Role ARN", appaws.Str(cr.Cluster.RoleArn))

	// Endpoint & Certificate Authority
	d.Section("Endpoint & Certificate")
	if endpoint := appaws.Str(cr.Cluster.Endpoint); endpoint != "" {
		d.Field("API Server Endpoint", endpoint)
	} else {
		d.Field("API Server Endpoint", render.NotConfigured)
	}
	if cr.Cluster.CertificateAuthority != nil && cr.Cluster.CertificateAuthority.Data != nil {
		d.Field("Certificate Authority", "Present")
	} else {
		d.Field("Certificate Authority", render.NotConfigured)
	}

	// VPC Configuration
	d.Section("VPC Configuration")
	if vpc := cr.Cluster.ResourcesVpcConfig; vpc != nil {
		d.Field("VPC ID", appaws.Str(vpc.VpcId))
		if len(vpc.SubnetIds) > 0 {
			d.Field("Subnets", strings.Join(vpc.SubnetIds, ", "))
		} else {
			d.Field("Subnets", render.Empty)
		}
		if len(vpc.SecurityGroupIds) > 0 {
			d.Field("Security Groups", strings.Join(vpc.SecurityGroupIds, ", "))
		} else {
			d.Field("Security Groups", render.Empty)
		}
		d.Field("Cluster Security Group", appaws.Str(vpc.ClusterSecurityGroupId))
		d.Field("Endpoint Public Access", formatBool(vpc.EndpointPublicAccess))
		d.Field("Endpoint Private Access", formatBool(vpc.EndpointPrivateAccess))
		if len(vpc.PublicAccessCidrs) > 0 {
			d.Field("Public Access CIDRs", strings.Join(vpc.PublicAccessCidrs, ", "))
		}
	}

	// Kubernetes Network Configuration
	d.Section("Kubernetes Network")
	if netCfg := cr.Cluster.KubernetesNetworkConfig; netCfg != nil {
		d.Field("IP Family", string(netCfg.IpFamily))
		d.Field("Service IPv4 CIDR", appaws.Str(netCfg.ServiceIpv4Cidr))
		if cidr := appaws.Str(netCfg.ServiceIpv6Cidr); cidr != "" {
			d.Field("Service IPv6 CIDR", cidr)
		}
		if lb := netCfg.ElasticLoadBalancing; lb != nil {
			d.Field("Elastic Load Balancing", formatBool(appaws.Bool(lb.Enabled)))
		}
	}

	// Logging
	d.Section("Logging")
	if logging := cr.Cluster.Logging; logging != nil && len(logging.ClusterLogging) > 0 {
		for _, log := range logging.ClusterLogging {
			if log.Enabled != nil && *log.Enabled {
				types := make([]string, len(log.Types))
				for i, t := range log.Types {
					types[i] = string(t)
				}
				d.Field("Enabled Log Types", strings.Join(types, ", "))
			} else {
				d.Field("Logging", "Disabled")
			}
		}
	} else {
		d.Field("Logging", "Disabled")
	}

	// Identity
	d.Section("Identity")
	if identity := cr.Cluster.Identity; identity != nil && identity.Oidc != nil {
		d.Field("OIDC Issuer", appaws.Str(identity.Oidc.Issuer))
	}

	// Access Configuration
	if accessCfg := cr.Cluster.AccessConfig; accessCfg != nil {
		d.Section("Access Configuration")
		d.Field("Authentication Mode", string(accessCfg.AuthenticationMode))
		if accessCfg.BootstrapClusterCreatorAdminPermissions != nil {
			d.Field("Bootstrap Creator Admin", formatBool(appaws.Bool(accessCfg.BootstrapClusterCreatorAdminPermissions)))
		}
	}

	// Encryption
	if len(cr.Cluster.EncryptionConfig) > 0 {
		d.Section("Encryption")
		for i, enc := range cr.Cluster.EncryptionConfig {
			if enc.Provider != nil && enc.Provider.KeyArn != nil {
				d.Field(fmt.Sprintf("Key ARN #%d", i+1), *enc.Provider.KeyArn)
				if len(enc.Resources) > 0 {
					d.Field(fmt.Sprintf("Resources #%d", i+1), strings.Join(enc.Resources, ", "))
				}
			}
		}
	}

	// Health
	if health := cr.Cluster.Health; health != nil && len(health.Issues) > 0 {
		d.Section("Health Issues")
		for i, issue := range health.Issues {
			d.Field(fmt.Sprintf("Issue #%d Code", i+1), string(issue.Code))
			if msg := appaws.Str(issue.Message); msg != "" {
				d.Field(fmt.Sprintf("Issue #%d Message", i+1), msg)
			}
			if len(issue.ResourceIds) > 0 {
				d.Field(fmt.Sprintf("Issue #%d Resources", i+1), strings.Join(issue.ResourceIds, ", "))
			}
		}
	}

	// Tags
	if len(cr.Cluster.Tags) > 0 {
		d.Section("Tags")
		d.Tags(cr.Cluster.Tags)
	}

	// Full Details
	d.Section("Full Details")
	if jsonBytes, err := json.MarshalIndent(cr.Cluster, "", "  "); err == nil {
		d.Line(string(jsonBytes))
	}

	return d.String()
}

func (rnd *ClusterRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cr, ok := resource.(*ClusterResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: cr.GetName()},
		{Label: "Version", Value: cr.Version()},
		{Label: "Status", Value: cr.Status()},
		{Label: "Created", Value: cr.CreatedAge()},
	}
}

func (rnd *ClusterRenderer) Navigations(resource dao.Resource) []render.Navigation {
	cr, ok := resource.(*ClusterResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:         "n",
			Label:       "Node Groups",
			Service:     "eks",
			Resource:    "node-groups",
			FilterField: "ClusterName",
			FilterValue: cr.GetName(),
		},
		{
			Key:         "f",
			Label:       "Fargate Profiles",
			Service:     "eks",
			Resource:    "fargate-profiles",
			FilterField: "ClusterName",
			FilterValue: cr.GetName(),
		},
		{
			Key:         "a",
			Label:       "Add-ons",
			Service:     "eks",
			Resource:    "addons",
			FilterField: "ClusterName",
			FilterValue: cr.GetName(),
		},
		{
			Key:         "e",
			Label:       "Access Entries",
			Service:     "eks",
			Resource:    "access-entries",
			FilterField: "ClusterName",
			FilterValue: cr.GetName(),
		},
	}
}
