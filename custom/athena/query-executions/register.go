package queryexecutions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("athena", "query-executions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewQueryExecutionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewQueryExecutionRenderer()
		},
	})
}
