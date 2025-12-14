package events

import "github.com/clawscli/claws/internal/registry"

func init() {
	registry.Global.RegisterCustom("cloudtrail", "events", registry.Entry{
		DAOFactory:      NewEventDAO,
		RendererFactory: NewEventRenderer,
	})
}
