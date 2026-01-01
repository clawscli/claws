package tgwattachments

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("vpc", "tgw-attachments", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewTGWAttachmentDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewTGWAttachmentRenderer()
		},
	})
}
