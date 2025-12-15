package httpapis

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("apigateway", "http-apis", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewHttpAPIDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewHttpAPIRenderer()
		},
	})
}
