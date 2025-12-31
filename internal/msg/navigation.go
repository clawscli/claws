// Package msg defines application-level messages for navigation and state changes.
// These messages are sent between views and handled by the app layer.
package msg

import "github.com/clawscli/claws/internal/config"

type ProfilesChangedMsg struct {
	Selections []config.ProfileSelection
}

type RegionChangedMsg struct {
	Regions []string
}
