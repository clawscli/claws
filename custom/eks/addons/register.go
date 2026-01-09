package addons

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("eks", "addons", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewAddonDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewAddonRenderer()
		},
	})
}
