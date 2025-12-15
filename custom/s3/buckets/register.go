package buckets

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("s3", "buckets", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewBucketDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewBucketRenderer()
		},
	})
}
