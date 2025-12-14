package activities

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("autoscaling", "activities", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewActivityDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewActivityRenderer()
		},
	})
}
