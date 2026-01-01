package elasticips

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("ec2", "elastic-ips", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewElasticIPDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewElasticIPRenderer()
		},
	})
}
