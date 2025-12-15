package rulegroups

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("network-firewall", "rule-groups", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRuleGroupDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRuleGroupRenderer()
		},
	})
}
