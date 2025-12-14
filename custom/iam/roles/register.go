package roles

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("iam", "roles", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRoleDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRoleRenderer()
		},
	})
}
