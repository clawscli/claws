package aws

import (
	"os"
	"strings"

	"github.com/clawscli/claws/internal/config"
)

// BuildSubprocessEnv constructs environment variables for AWS CLI subprocesses.
// Handles profile selection and region injection based on credential mode:
//
//   - SDKDefault: preserve existing AWS_PROFILE (don't modify)
//   - EnvOnly: remove AWS_PROFILE, set config/credentials files to /dev/null
//   - NamedProfile: set AWS_PROFILE to the profile name
//
// Region behavior:
//   - If region is non-empty, inject both AWS_REGION and AWS_DEFAULT_REGION
//   - If region is empty, don't modify existing region env vars
func BuildSubprocessEnv(baseEnv []string, sel config.ProfileSelection, region string) []string {
	if baseEnv == nil {
		baseEnv = os.Environ()
	}

	// Build set of keys to remove
	keysToRemove := map[string]bool{}

	switch sel.Mode {
	case config.ModeEnvOnly, config.ModeNamedProfile:
		keysToRemove["AWS_PROFILE"] = true
	}

	if region != "" {
		keysToRemove["AWS_REGION"] = true
		keysToRemove["AWS_DEFAULT_REGION"] = true
	}

	// Filter and rebuild env
	env := make([]string, 0, len(baseEnv)+4)
	for _, e := range baseEnv {
		keep := true
		for key := range keysToRemove {
			if strings.HasPrefix(e, key+"=") {
				keep = false
				break
			}
		}
		if keep {
			env = append(env, e)
		}
	}

	// Add profile-related env vars based on mode
	switch sel.Mode {
	case config.ModeNamedProfile:
		env = append(env, "AWS_PROFILE="+sel.ProfileName)
	case config.ModeEnvOnly:
		// Force CLI to ignore config files, use IMDS/env only
		env = append(env, "AWS_CONFIG_FILE="+os.DevNull)
		env = append(env, "AWS_SHARED_CREDENTIALS_FILE="+os.DevNull)
	}

	// Add region if set
	if region != "" {
		env = append(env, "AWS_REGION="+region)
		env = append(env, "AWS_DEFAULT_REGION="+region)
	}

	return env
}
