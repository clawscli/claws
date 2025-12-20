package profile

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("local", "profile", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewProfileDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewProfileRenderer()
		},
	})
}
