package vpcendpoints

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("vpc", "endpoints", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewVpcEndpointDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewVpcEndpointRenderer()
		},
	})
}
