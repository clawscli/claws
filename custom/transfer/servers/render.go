package servers

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ServerRenderer renders Transfer Family servers.
// Ensure ServerRenderer implements render.Navigator
var _ render.Navigator = (*ServerRenderer)(nil)

type ServerRenderer struct {
	render.BaseRenderer
}

// NewServerRenderer creates a new ServerRenderer.
func NewServerRenderer() render.Renderer {
	return &ServerRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "transfer",
			Resource: "servers",
			Cols: []render.Column{
				{Name: "SERVER ID", Width: 24, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "ENDPOINT TYPE", Width: 14, Getter: getEndpointType},
				{Name: "DOMAIN", Width: 10, Getter: getDomain},
				{Name: "IDENTITY", Width: 18, Getter: getIdentityProvider},
				{Name: "USERS", Width: 8, Getter: getUserCount},
			},
		},
	}
}

func getState(r dao.Resource) string {
	srv, ok := r.(*ServerResource)
	if !ok {
		return ""
	}
	return srv.State()
}

func getEndpointType(r dao.Resource) string {
	srv, ok := r.(*ServerResource)
	if !ok {
		return ""
	}
	return srv.EndpointType()
}

func getDomain(r dao.Resource) string {
	srv, ok := r.(*ServerResource)
	if !ok {
		return ""
	}
	return srv.Domain()
}

func getIdentityProvider(r dao.Resource) string {
	srv, ok := r.(*ServerResource)
	if !ok {
		return ""
	}
	return srv.IdentityProviderType()
}

func getUserCount(r dao.Resource) string {
	srv, ok := r.(*ServerResource)
	if !ok {
		return ""
	}
	count := srv.UserCount()
	if count == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", count)
}

// RenderDetail renders the detail view for a Transfer Family server.
func (r *ServerRenderer) RenderDetail(resource dao.Resource) string {
	srv, ok := resource.(*ServerResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Transfer Family Server", srv.ServerId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Server ID", srv.ServerId())
	d.Field("ARN", srv.GetARN())
	d.Field("State", srv.State())
	d.Field("Endpoint Type", srv.EndpointType())
	d.Field("Domain", srv.Domain())

	// Protocols
	if protocols := srv.ProtocolsString(); protocols != "" {
		d.Field("Protocols", protocols)
	}

	// Identity Provider
	d.Section("Identity Provider")
	d.Field("Type", srv.IdentityProviderType())
	if idp := srv.IdentityProviderDetails(); idp != nil {
		if idp.Url != nil {
			d.Field("URL", *idp.Url)
		}
		if idp.InvocationRole != nil {
			d.Field("Invocation Role", *idp.InvocationRole)
		}
		if idp.DirectoryId != nil {
			d.Field("Directory ID", *idp.DirectoryId)
		}
		if idp.Function != nil {
			d.Field("Function ARN", *idp.Function)
		}
		if idp.SftpAuthenticationMethods != "" {
			d.Field("SFTP Auth Methods", string(idp.SftpAuthenticationMethods))
		}
	}

	// Security
	d.Section("Security")
	if policy := srv.SecurityPolicyName(); policy != "" {
		d.Field("Security Policy", policy)
	}
	if cert := srv.Certificate(); cert != "" {
		d.Field("Certificate ARN", cert)
	}
	if fingerprint := srv.HostKeyFingerprint(); fingerprint != "" {
		d.Field("Host Key Fingerprint", fingerprint)
	}

	// Protocol Details
	if pd := srv.ProtocolDetails(); pd != nil {
		d.Section("Protocol Configuration")
		if pd.PassiveIp != nil {
			d.Field("Passive IP", *pd.PassiveIp)
		}
		if pd.SetStatOption != "" {
			d.Field("SetStat Option", string(pd.SetStatOption))
		}
		if pd.TlsSessionResumptionMode != "" {
			d.Field("TLS Session Resumption", string(pd.TlsSessionResumptionMode))
		}
		if len(pd.As2Transports) > 0 {
			transports := ""
			for i, t := range pd.As2Transports {
				if i > 0 {
					transports += ", "
				}
				transports += string(t)
			}
			d.Field("AS2 Transports", transports)
		}
	}

	// AS2 Egress IPs
	if ips := srv.As2ServiceManagedEgressIpAddresses(); len(ips) > 0 {
		d.Field("AS2 Egress IPs", strings.Join(ips, ", "))
	}

	// Workflow Details
	if wf := srv.WorkflowDetails(); wf != nil {
		hasWorkflows := false
		if len(wf.OnUpload) > 0 || len(wf.OnPartialUpload) > 0 {
			d.Section("Workflows")
			hasWorkflows = true
		}
		if hasWorkflows {
			for _, w := range wf.OnUpload {
				if w.WorkflowId != nil {
					d.Field("On Upload", *w.WorkflowId)
				}
			}
			for _, w := range wf.OnPartialUpload {
				if w.WorkflowId != nil {
					d.Field("On Partial Upload", *w.WorkflowId)
				}
			}
		}
	}

	// Endpoint Details (for VPC)
	if srv.EndpointType() == "VPC" || srv.EndpointType() == "VPC_ENDPOINT" {
		d.Section("VPC Configuration")
		if vpcId := srv.VpcId(); vpcId != "" {
			d.Field("VPC ID", vpcId)
		}
		if endpoint := srv.Endpoint(); endpoint != "" {
			d.Field("VPC Endpoint ID", endpoint)
		}
		if subnets := srv.SubnetIds(); len(subnets) > 0 {
			d.Field("Subnet IDs", strings.Join(subnets, ", "))
		}
		if sgs := srv.SecurityGroupIds(); len(sgs) > 0 {
			d.Field("Security Group IDs", strings.Join(sgs, ", "))
		}
		if allocIds := srv.AddressAllocationIds(); len(allocIds) > 0 {
			d.Field("Address Allocation IDs", strings.Join(allocIds, ", "))
		}
	}

	// Logging
	d.Section("Logging")
	if role := srv.LoggingRole(); role != "" {
		d.Field("Logging Role ARN", role)
	} else {
		d.Field("Logging Role", "(not configured)")
	}
	if dests := srv.StructuredLogDestinations(); len(dests) > 0 {
		d.Field("Log Destinations", strings.Join(dests, ", "))
	}

	// S3 Options
	if s3Opts := srv.S3StorageOptions(); s3Opts != "" {
		d.Section("S3 Storage Options")
		d.Field("Directory Listing Optimization", s3Opts)
	}

	// Login Banners
	if banner := srv.PreAuthenticationLoginBanner(); banner != "" {
		d.Section("Login Banners")
		d.Field("Pre-Authentication", banner)
	}
	if banner := srv.PostAuthenticationLoginBanner(); banner != "" {
		if srv.PreAuthenticationLoginBanner() == "" {
			d.Section("Login Banners")
		}
		d.Field("Post-Authentication", banner)
	}

	// Tags
	if tags := srv.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for _, tag := range tags {
			if tag.Key != nil && tag.Value != nil {
				d.Field(*tag.Key, *tag.Value)
			}
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a Transfer Family server.
func (r *ServerRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	srv, ok := resource.(*ServerResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Server ID", Value: srv.ServerId()},
		{Label: "ARN", Value: srv.GetARN()},
		{Label: "State", Value: srv.State()},
		{Label: "Endpoint Type", Value: srv.EndpointType()},
	}

	if protocols := srv.ProtocolsString(); protocols != "" {
		fields = append(fields, render.SummaryField{Label: "Protocols", Value: protocols})
	}

	return fields
}

// Navigations returns available navigations from a Transfer Family server.
func (r *ServerRenderer) Navigations(resource dao.Resource) []render.Navigation {
	srv, ok := resource.(*ServerResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "u",
			Label:       "Users",
			Service:     "transfer",
			Resource:    "users",
			FilterField: "ServerId",
			FilterValue: srv.ServerId(),
		},
	}
}
