package loggroups

import (
	"github.com/clawscli/claws/internal/action"
)

func init() {
	action.Global.Register("cloudwatch", "log-groups", []action.Action{
		{
			Name:     "Tail Logs",
			Shortcut: "t",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${ID}" --since 1h --follow`,
		},
		{
			Name:     "View Recent (1h)",
			Shortcut: "1",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${ID}" --since 1h | less -R`,
		},
		{
			Name:     "View Recent (24h)",
			Shortcut: "2",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${ID}" --since 24h | less -R`,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Confirm:   true,
			Dangerous: true,
		},
	})
}
