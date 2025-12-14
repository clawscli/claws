package functions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("lambda", "functions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewFunctionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewFunctionRenderer()
		},
	})
}
