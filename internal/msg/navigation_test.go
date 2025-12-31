package msg

import (
	"testing"

	"github.com/clawscli/claws/internal/config"
)

func TestProfilesChangedMsg(t *testing.T) {
	sel := config.NamedProfile("production")
	msg := ProfilesChangedMsg{Selections: []config.ProfileSelection{sel}}

	if len(msg.Selections) != 1 {
		t.Errorf("len(Selections) = %d, want 1", len(msg.Selections))
	}
	if !msg.Selections[0].IsNamedProfile() {
		t.Error("expected IsNamedProfile() to be true")
	}
	if msg.Selections[0].ProfileName != "production" {
		t.Errorf("ProfileName = %q, want %q", msg.Selections[0].ProfileName, "production")
	}
}

func TestRegionChangedMsg(t *testing.T) {
	msg := RegionChangedMsg{Regions: []string{"us-west-2", "ap-northeast-1"}}

	if len(msg.Regions) != 2 {
		t.Errorf("len(Regions) = %d, want 2", len(msg.Regions))
	}
	if msg.Regions[0] != "us-west-2" {
		t.Errorf("Regions[0] = %q, want %q", msg.Regions[0], "us-west-2")
	}
}
