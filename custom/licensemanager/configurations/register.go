package configurations

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("license-manager", "configurations", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewConfigurationDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewConfigurationRenderer()
		},
	})
}
