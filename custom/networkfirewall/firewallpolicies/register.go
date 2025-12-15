package firewallpolicies

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("network-firewall", "firewall-policies", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewFirewallPolicyDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewFirewallPolicyRenderer()
		},
	})
}
