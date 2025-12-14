package users

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("iam", "users", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewUserDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewUserRenderer()
		},
	})
}
