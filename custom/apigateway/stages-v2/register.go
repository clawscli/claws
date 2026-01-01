package stagesv2

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("apigateway", "stages-v2", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewStageV2DAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewStageV2Renderer()
		},
	})
}
