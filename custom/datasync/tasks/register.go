package tasks

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("datasync", "tasks", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTaskDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTaskRenderer()
		},
	})
}
