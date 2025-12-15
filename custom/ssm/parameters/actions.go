package parameters

import (
	"github.com/clawscli/claws/internal/action"
)

func init() {
	action.Global.Register("ssm", "parameters", []action.Action{
		{
			Name:     "View Value",
			Shortcut: "v",
			Type:     action.ActionTypeExec,
			Command:  `aws ssm get-parameter --name "${ID}" --with-decryption --query 'Parameter.Value' --output text | less -R`,
		},
		{
			Name:     "View History",
			Shortcut: "h",
			Type:     action.ActionTypeExec,
			Command:  `aws ssm get-parameter-history --name "${ID}" --with-decryption | less -R`,
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
