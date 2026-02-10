package builds

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("gamelift", "builds", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewBuildDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewBuildRenderer()
		},
	})
}
