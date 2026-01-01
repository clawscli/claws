package jobqueues

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("batch", "job-queues", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewJobQueueDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewJobQueueRenderer()
		},
	})
}
