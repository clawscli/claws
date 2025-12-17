package restorejobs

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("backup", "restore-jobs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRestoreJobDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRestoreJobRenderer()
		},
	})
}
