package routetables

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("vpc", "route-tables", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRouteTableDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRouteTableRenderer()
		},
	})
}
