package profile

import (
	"context"
	"fmt"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/view"
)

// Operation constants for profile actions
const (
	OperationSwitchProfile = "SwitchProfile"
)

func init() {
	action.Global.Register("local", "profile", []action.Action{
		{
			Name:      "Switch",
			Shortcut:  "s",
			Type:      action.ActionTypeAPI,
			Operation: OperationSwitchProfile,
		},
		{
			Name:     action.ActionNameSSOLogin,
			Shortcut: "l",
			Type:     action.ActionTypeExec,
			Command:  "aws sso login --profile ${NAME}",
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

	name := pr.Data.Name
	cfg := config.Global()

	// Determine selection based on resource name
	var sel config.ProfileSelection
	var msg string

	switch name {
	case config.SDKDefault().DisplayName():
		sel = config.SDKDefault()
		msg = "Using SDK default credentials"
	case config.EnvOnly().DisplayName():
		sel = config.EnvOnly()
		msg = "Using environment/IMDS credentials (ignoring ~/.aws config)"
	default:
		sel = config.NamedProfile(name)
		msg = fmt.Sprintf("Switched to profile: %s", name)
	}

	cfg.SetSelection(sel)

	return action.ActionResult{
		Success:     true,
		Message:     msg,
		FollowUpMsg: view.ProfileChangedMsg{Selection: sel},
	}
}
