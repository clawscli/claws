package topics

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("sns", "topics", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTopicDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTopicRenderer()
		},
	})
}
