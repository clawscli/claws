package keypairs

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("ec2", "key-pairs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewKeyPairDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewKeyPairRenderer()
		},
	})
}
