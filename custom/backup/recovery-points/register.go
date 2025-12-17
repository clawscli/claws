package recoverypoints

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("backup", "recovery-points", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewRecoveryPointDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewRecoveryPointRenderer()
		},
	})
}
