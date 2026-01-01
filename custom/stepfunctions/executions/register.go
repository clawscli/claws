package executions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("stepfunctions", "executions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewExecutionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewExecutionRenderer()
		},
	})
}
