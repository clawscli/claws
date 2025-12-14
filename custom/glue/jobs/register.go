package jobs

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("glue", "jobs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewJobDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewJobRenderer()
		},
	})
}
