package protectedresources

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("backup", "protected-resources", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewProtectedResourceDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewProtectedResourceRenderer()
		},
	})
}
