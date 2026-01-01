package knowledgebases

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock-agent", "knowledge-bases", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewKnowledgeBaseDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewKnowledgeBaseRenderer()
		},
	})
}
