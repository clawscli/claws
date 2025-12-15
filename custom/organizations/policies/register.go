package policies

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("organizations", "policies", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewPolicyDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewPolicyRenderer()
		},
	})
}
