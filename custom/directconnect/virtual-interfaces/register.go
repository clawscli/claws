package virtualinterfaces

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("directconnect", "virtual-interfaces", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewVirtualInterfaceDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewVirtualInterfaceRenderer()
		},
	})
}
