package firewalls

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("network-firewall", "firewalls", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewFirewallDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewFirewallRenderer()
		},
	})
}
