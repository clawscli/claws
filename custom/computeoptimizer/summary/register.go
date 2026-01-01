package summary

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("compute-optimizer", "summary", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewSummaryDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewSummaryRenderer()
		},
	})
}
