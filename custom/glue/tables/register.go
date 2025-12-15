package tables

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("glue", "tables", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTableDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTableRenderer()
		},
	})
}
