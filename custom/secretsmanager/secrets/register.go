package secrets

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("secretsmanager", "secrets", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewSecretDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewSecretRenderer()
		},
	})
}
