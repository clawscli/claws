package runtimes

import (
	"github.com/clawscli/claws/internal/action"
)

func init() {
	action.Global.Register("bedrock-agentcore", "runtimes", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Confirm:   true,
			Dangerous: true,
		},
	})
}
