package trainingjobs

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("sagemaker", "training-jobs", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTrainingJobDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTrainingJobRenderer()
		},
	})
}
