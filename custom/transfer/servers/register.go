package servers

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("transfer", "servers", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewServerDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewServerRenderer()
		},
	})
}
