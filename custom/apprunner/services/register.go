package services

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("apprunner", "services", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewServiceDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewServiceRenderer()
		},
	})
}
