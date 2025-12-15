package savingsplans

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("risp", "savings-plans", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewSavingsPlanDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewSavingsPlanRenderer()
		},
	})
}
