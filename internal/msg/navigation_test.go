package msg

import (
	"testing"

	"github.com/clawscli/claws/internal/config"
)

func TestProfileChangedMsg(t *testing.T) {
	sel := config.NamedProfile("production")
	msg := ProfileChangedMsg{Selection: sel}

	if !msg.Selection.IsNamedProfile() {
		t.Error("expected IsNamedProfile() to be true")
	}
	if msg.Selection.ProfileName != "production" {
		t.Errorf("ProfileName = %q, want %q", msg.Selection.ProfileName, "production")
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
