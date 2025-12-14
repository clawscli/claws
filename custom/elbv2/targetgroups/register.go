package targetgroups

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("elbv2", "target-groups", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTargetGroupDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTargetGroupRenderer()
		},
	})
}
