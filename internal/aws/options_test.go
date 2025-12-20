package aws

import (
	"testing"

	"github.com/clawscli/claws/internal/config"
)

func TestSelectionLoadOptions(t *testing.T) {
	tests := []struct {
		name    string
		sel     config.ProfileSelection
		wantLen int
	}{
		{
			name:    "SDK default",
			sel:     config.SDKDefault(),
			wantLen: 1, // just IMDS region
		},
		{
			name:    "env only",
			sel:     config.EnvOnly(),
			wantLen: 3, // IMDS region + 2 empty file options
		},
		{
			name:    "named profile",
			sel:     config.NamedProfile("production"),
			wantLen: 2, // IMDS region + profile option
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := SelectionLoadOptions(tt.sel)
			if len(opts) != tt.wantLen {
				t.Errorf("SelectionLoadOptions(%v) returned %d options, want %d", tt.sel, len(opts), tt.wantLen)
			}
		})
	}
}
