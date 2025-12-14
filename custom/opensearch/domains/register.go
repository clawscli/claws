package domains

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("opensearch", "domains", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewDomainDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewDomainRenderer()
		},
	})
}
