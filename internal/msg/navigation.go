// Package msg defines application-level messages for navigation and state changes.
// These messages are sent between views and handled by the app layer.
package msg

import "github.com/clawscli/claws/internal/config"

// ProfileChangedMsg is sent when profile is changed.
// Handled by app.go to refresh views with new credentials.
type ProfileChangedMsg struct {
	Selection config.ProfileSelection
}

// RegionChangedMsg is sent when region is changed.
// Handled by app.go to refresh views with new region.
type RegionChangedMsg struct {
	Regions []string
}
