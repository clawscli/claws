package targets

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("elbv2", "targets", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTargetDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTargetRenderer()
		},
	})
}
