package gamesessions

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("gamelift", "game-sessions", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewGameSessionDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewGameSessionRenderer()
		},
	})
}
