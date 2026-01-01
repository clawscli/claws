package inferenceprofiles

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("bedrock", "inference-profiles", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewInferenceProfileDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewInferenceProfileRenderer()
		},
	})
}
