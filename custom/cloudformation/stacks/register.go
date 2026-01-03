package stacks

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("cloudformation", "stacks", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewStackDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewStackRenderer()
		},
	})
}
