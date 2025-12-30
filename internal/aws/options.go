package aws

import (
	"github.com/aws/aws-sdk-go-v2/config"

	appconfig "github.com/clawscli/claws/internal/config"
)

// SelectionLoadOptions returns config load options based on the given ProfileSelection.
// This centralizes the logic for handling different credential modes:
//   - ModeSDKDefault: no extra options, let SDK use standard chain
//   - ModeEnvOnly: ignore ~/.aws files, use IMDS/environment only
//   - ModeNamedProfile: explicitly use that profile from ~/.aws files
func SelectionLoadOptions(sel appconfig.ProfileSelection) []func(*config.LoadOptions) error {
	opts := []func(*config.LoadOptions) error{
		config.WithEC2IMDSRegion(),
	}
	switch sel.Mode {
	case appconfig.ModeEnvOnly:
		opts = append(opts,
			config.WithSharedConfigFiles([]string{}),
			config.WithSharedCredentialsFiles([]string{}),
		)
	case appconfig.ModeNamedProfile:
		opts = append(opts, config.WithSharedConfigProfile(sel.ProfileName))
	case appconfig.ModeSDKDefault:
		// No extra options - let SDK use standard chain
	}
	return opts
}
