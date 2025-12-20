package profile

import (
	"context"
	"fmt"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/view"
)

// Action names for profile operations (used by action_menu for post-exec handling)
const (
	ActionNameSSOLogin     = "SSO Login"
	ActionNameConsoleLogin = "Console Login"
)

func init() {
	action.Global.Register("local", "profile", []action.Action{
		{
			Name:      "Switch",
			Shortcut:  "s",
			Type:      action.ActionTypeAPI,
			Operation: "SwitchProfile",
		},
		{
			Name:     ActionNameSSOLogin,
			Shortcut: "l",
			Type:     action.ActionTypeExec,
			Command:  "aws sso login --profile ${NAME}",
		},
		{
			Name:     ActionNameConsoleLogin,
			Shortcut: "c",
			Type:     action.ActionTypeExec,
			Command:  "aws login --remote",
		},
	})

	action.RegisterExecutor("local", "profile", executeProfileAction)
}

func executeProfileAction(_ context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "SwitchProfile":
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

	profileName := pr.Data.Name

	// Handle (Environment) - use environment credentials, ignore ~/.aws config
	if profileName == config.EnvironmentCredentialsDisplayName {
		config.Global().SetProfile(config.UseEnvironmentCredentials)
		return action.ActionResult{
			Success:     true,
			Message:     "Using environment credentials (ignoring ~/.aws config)",
			FollowUpMsg: view.ProfileChangedMsg{Profile: config.UseEnvironmentCredentials},
		}
	}

	config.Global().SetProfile(profileName)

	return action.ActionResult{
		Success:     true,
		Message:     fmt.Sprintf("Switched to profile: %s", profileName),
		FollowUpMsg: view.ProfileChangedMsg{Profile: profileName},
	}
}
