package rules

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("eventbridge", "rules", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRuleDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRuleRenderer()
		},
	})
}
