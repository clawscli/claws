package distributions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("cloudfront", "distributions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewDistributionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewDistributionRenderer()
		},
	})
}
