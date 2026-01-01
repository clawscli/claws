package runtimes

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock-agentcore", "runtimes", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRuntimeDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRuntimeRenderer()
		},
	})
}
