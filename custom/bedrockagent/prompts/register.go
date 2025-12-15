package prompts

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock-agent", "prompts", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewPromptDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewPromptRenderer()
		},
	})
}
