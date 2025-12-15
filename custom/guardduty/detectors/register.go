package detectors

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("guardduty", "detectors", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewDetectorDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewDetectorRenderer()
		},
	})
}
