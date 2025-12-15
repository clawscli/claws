package clusters

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("emr", "clusters", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewClusterDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewClusterRenderer()
		},
	})
}
