package findings

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("guardduty", "findings", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewFindingDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewFindingRenderer()
		},
	})
}
