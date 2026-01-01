package versions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock-agentcore", "versions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewVersionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewVersionRenderer()
		},
	})
}
