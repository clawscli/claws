package foundationmodels

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock", "foundation-models", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewFoundationModelDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewFoundationModelRenderer()
		},
	})
}
