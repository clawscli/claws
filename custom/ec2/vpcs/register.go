package vpcs

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("vpc", "vpcs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewVPCDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewVPCRenderer()
		},
	})
}
