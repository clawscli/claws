package analyzers

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("accessanalyzer", "analyzers", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewAnalyzerDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewAnalyzerRenderer()
		},
	})
}
