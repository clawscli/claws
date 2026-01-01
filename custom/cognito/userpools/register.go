package userpools

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("cognito-idp", "user-pools", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewUserPoolDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewUserPoolRenderer()
		},
	})
}
