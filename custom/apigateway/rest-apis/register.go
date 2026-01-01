package restapis

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("apigateway", "rest-apis", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRestAPIDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRestAPIRenderer()
		},
	})
}
