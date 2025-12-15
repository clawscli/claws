package costs

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("costexplorer", "costs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewCostDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewCostRenderer()
		},
	})
}
