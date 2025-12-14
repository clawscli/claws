package instances

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("rds", "instances", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewInstanceDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewInstanceRenderer()
		},
	})
}
