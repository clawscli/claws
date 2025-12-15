package capacityreservations

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("ec2", "capacity-reservations", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewCapacityReservationDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewCapacityReservationRenderer()
		},
	})
}
