package repositories

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("ecr", "repositories", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRepositoryDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRepositoryRenderer()
		},
	})
}
