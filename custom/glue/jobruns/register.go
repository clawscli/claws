package jobruns

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("glue", "job-runs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewJobRunDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewJobRunRenderer()
		},
	})
}
