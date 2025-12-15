package steps

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("emr", "steps", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewStepDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewStepRenderer()
		},
	})
}
