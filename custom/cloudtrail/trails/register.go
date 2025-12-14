package trails

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("cloudtrail", "trails", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTrailDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTrailRenderer()
		},
	})
}
