package profile

import (
	"context"
	"fmt"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/msg"
)

// Operation constants for profile actions
const (
	OperationSwitchProfile = "SwitchProfile"
)

// isSSOProfile returns true if the resource is a profile with SSO configuration
func isSSOProfile(r dao.Resource) bool {
	pr, ok := r.(*ProfileResource)
	if !ok {
		return false
	}
	return pr.Data.SSOStartURL != "" || pr.Data.SSOSession != ""
}

func init() {
	action.Global.Register("local", "profile", []action.Action{
		{
			Name:      "Switch",
			Shortcut:  "s",
			Type:      action.ActionTypeAPI,
			Operation: OperationSwitchProfile,
		},
		{
			Name:       action.ActionNameSSOLogin,
			Shortcut:   "l",
			Type:       action.ActionTypeExec,
			Command:    "aws sso login --profile ${NAME}",
			SkipAWSEnv: true, // Must access ~/.aws files directly
			Filter:     isSSOProfile,
			PostExecFollowUp: func(r dao.Resource) any {
				sel := config.ProfileSelectionFromID(r.GetID())
				config.Global().SetSelection(sel)
				return msg.ProfileChangedMsg{Selection: sel}
			},
		},
	})

	action.RegisterExecutor("local", "profile", executeProfileAction)
}

func executeProfileAction(_ context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case OperationSwitchProfile:
		return executeSwitchProfile(resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeSwitchProfile(resource dao.Resource) action.ActionResult {
	pr, ok := resource.(*ProfileResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	sel := config.ProfileSelectionFromID(pr.GetID())
	config.Global().SetSelection(sel)

	resultMsg := fmt.Sprintf("Switched to profile: %s", sel.DisplayName())

	return action.ActionResult{
		Success:     true,
		Message:     resultMsg,
		FollowUpMsg: msg.ProfileChangedMsg{Selection: sel},
	}
}
