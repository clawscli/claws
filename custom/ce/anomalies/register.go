package anomalies

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("ce", "anomalies", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewAnomalyDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewAnomalyRenderer()
		},
	})
}
