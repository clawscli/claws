package jobdefinitions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("batch", "job-definitions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewJobDefinitionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewJobDefinitionRenderer()
		},
	})
}
