package volumes

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("ec2", "volumes", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewVolumeDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewVolumeRenderer()
		},
	})
}
