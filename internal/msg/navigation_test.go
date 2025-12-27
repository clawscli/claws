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
	msg := RegionChangedMsg{Region: "us-west-2"}

	if msg.Region != "us-west-2" {
		t.Errorf("Region = %q, want %q", msg.Region, "us-west-2")
	}
}
