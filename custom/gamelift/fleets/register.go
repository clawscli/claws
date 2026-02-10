package fleets

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("gamelift", "fleets", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewFleetDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewFleetRenderer()
		},
	})
}
