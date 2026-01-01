package instanceprofiles

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("iam", "instance-profiles", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewInstanceProfileDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewInstanceProfileRenderer()
		},
	})
}
