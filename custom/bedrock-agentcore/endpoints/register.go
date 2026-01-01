package endpoints

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock-agentcore", "endpoints", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewEndpointDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewEndpointRenderer()
		},
	})
}
