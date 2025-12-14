package internetgateways

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("vpc", "internet-gateways", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewInternetGatewayDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewInternetGatewayRenderer()
		},
	})
}
