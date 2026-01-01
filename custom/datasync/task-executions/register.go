package taskexecutions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("datasync", "task-executions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTaskExecutionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTaskExecutionRenderer()
		},
	})
}
