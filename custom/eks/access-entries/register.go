package accessentries

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("eks", "access-entries", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewAccessEntryDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewAccessEntryRenderer()
		},
	})
}
