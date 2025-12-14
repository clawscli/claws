package quotas

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("service-quotas", "quotas", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewQuotaDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewQuotaRenderer()
		},
	})
}
