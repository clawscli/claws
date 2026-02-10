package gamesessionqueues

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("gamelift", "game-session-queues", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewQueueDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewQueueRenderer()
		},
	})
}
