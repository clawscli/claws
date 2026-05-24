package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
)

const (
	ssoDeviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"
	ssoRefreshGrantType    = "refresh_token"
	ssoDefaultScope        = "sso:account:access"
)

// SSOLoginResult describes what the SSO login/refresh operation did.
type SSOLoginResult struct {
	Message   string
	ExpiresAt time.Time
}

type ssoCachedToken struct {
	StartURL              string `json:"startUrl,omitempty"`
	Region                string `json:"region,omitempty"`
	AccessToken           string `json:"accessToken,omitempty"`
	ExpiresAt             string `json:"expiresAt,omitempty"`
	RefreshToken          string `json:"refreshToken,omitempty"`
	ClientID              string `json:"clientId,omitempty"`
	ClientSecret          string `json:"clientSecret,omitempty"`
	RegistrationExpiresAt string `json:"registrationExpiresAt,omitempty"`
}

// RunSSOLogin ensures the selected IAM Identity Center profile has a usable SSO session.
// It reuses and refreshes the standard AWS SSO cache when possible, and starts a
// device authorization flow only when no usable cached token is available.
func RunSSOLogin(ctx context.Context, profile ProfileInfo, out io.Writer) (SSOLoginResult, error) {
	if out == nil {
		out = io.Discard
	}
	if err := validateSSOProfile(profile); err != nil {
		return SSOLoginResult{}, err
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(profile.SSORegion))
	if err != nil {
		return SSOLoginResult{}, fmt.Errorf("load SSO OIDC config: %w", err)
	}

	tokenPath, err := ssoTokenCachePath(profile)
	if err != nil {
		return SSOLoginResult{}, err
	}

	if expiresAt, err := retrieveSSORoleCredentials(ctx, cfg, profile, tokenPath); err == nil {
		return SSOLoginResult{
			Message:   "SSO session ready",
			ExpiresAt: expiresAt,
		}, nil
	}

	if err := startSSODeviceLogin(ctx, cfg, profile, tokenPath, out); err != nil {
		return SSOLoginResult{}, err
	}

	expiresAt, err := retrieveSSORoleCredentials(ctx, cfg, profile, tokenPath)
	if err != nil {
		return SSOLoginResult{}, fmt.Errorf("validate SSO role credentials: %w", err)
	}
	return SSOLoginResult{
		Message:   "SSO login successful",
		ExpiresAt: expiresAt,
	}, nil
}

func validateSSOProfile(profile ProfileInfo) error {
	missing := make([]string, 0, 4)
	if profile.SSOStartURL == "" {
		missing = append(missing, "sso_start_url")
	}
	if profile.SSORegion == "" {
		missing = append(missing, "sso_region")
	}
	if profile.SSOAccountID == "" {
		missing = append(missing, "sso_account_id")
	}
	if profile.SSORoleName == "" {
		missing = append(missing, "sso_role_name")
	}
	if len(missing) > 0 {
		return fmt.Errorf("profile %q is missing SSO settings: %s", profile.Name, strings.Join(missing, ", "))
	}
	return nil
}

func ssoTokenCachePath(profile ProfileInfo) (string, error) {
	key := profile.SSOStartURL
	if profile.SSOSession != "" {
		key = profile.SSOSession
	}
	path, err := ssocreds.StandardCachedTokenFilepath(key)
	if err != nil {
		return "", fmt.Errorf("resolve SSO token cache path: %w", err)
	}
	return path, nil
}

func retrieveSSORoleCredentials(ctx context.Context, cfg awssdk.Config, profile ProfileInfo, tokenPath string) (time.Time, error) {
	ssoClient := sso.NewFromConfig(cfg)
	ssoOIDCClient := ssooidc.NewFromConfig(cfg)
	provider := ssocreds.New(ssoClient, profile.SSOAccountID, profile.SSORoleName, profile.SSOStartURL, func(options *ssocreds.Options) {
		options.CachedTokenFilepath = tokenPath
		options.SSOTokenProvider = ssocreds.NewSSOTokenProvider(ssoOIDCClient, tokenPath)
	})
	credentials, err := awssdk.NewCredentialsCache(provider).Retrieve(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return credentials.Expires, nil
}

func startSSODeviceLogin(ctx context.Context, cfg awssdk.Config, profile ProfileInfo, tokenPath string, out io.Writer) error {
	client := ssooidc.NewFromConfig(cfg)
	registerOutput, err := client.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: awssdk.String("claws"),
		ClientType: awssdk.String("public"),
		GrantTypes: []string{ssoDeviceCodeGrantType, ssoRefreshGrantType},
		Scopes:     ssoScopes(profile),
	})
	if err != nil {
		return fmt.Errorf("register SSO OIDC client: %w", err)
	}

	deviceOutput, err := client.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerOutput.ClientId,
		ClientSecret: registerOutput.ClientSecret,
		StartUrl:     awssdk.String(profile.SSOStartURL),
	})
	if err != nil {
		return fmt.Errorf("start SSO device authorization: %w", err)
	}

	writeDeviceInstructions(out, deviceOutput)
	if uri := awssdk.ToString(deviceOutput.VerificationUriComplete); uri != "" {
		if err := openBrowser(uri); err == nil {
			_, _ = fmt.Fprintln(out, "Opened the authorization URL in your browser.")
		}
	}

	createOutput, err := pollSSODeviceToken(ctx, client, registerOutput, deviceOutput)
	if err != nil {
		return err
	}
	if err := storeSSOToken(tokenPath, profile, registerOutput, createOutput); err != nil {
		return err
	}
	return nil
}

func ssoScopes(profile ProfileInfo) []string {
	if profile.SSOScopes == "" {
		return []string{ssoDefaultScope}
	}
	fields := strings.FieldsFunc(profile.SSOScopes, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	scopes := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			scopes = append(scopes, field)
		}
	}
	if len(scopes) == 0 {
		return []string{ssoDefaultScope}
	}
	return scopes
}

func writeDeviceInstructions(out io.Writer, deviceOutput *ssooidc.StartDeviceAuthorizationOutput) {
	_, _ = fmt.Fprintln(out, "Complete AWS SSO authorization in your browser.")
	if uri := awssdk.ToString(deviceOutput.VerificationUriComplete); uri != "" {
		_, _ = fmt.Fprintf(out, "URL: %s\n", uri)
	} else if uri := awssdk.ToString(deviceOutput.VerificationUri); uri != "" {
		_, _ = fmt.Fprintf(out, "URL: %s\n", uri)
	}
	if code := awssdk.ToString(deviceOutput.UserCode); code != "" {
		_, _ = fmt.Fprintf(out, "Code: %s\n", code)
	}
	_, _ = fmt.Fprintln(out, "Waiting for authorization...")
}

func pollSSODeviceToken(ctx context.Context, client *ssooidc.Client, registerOutput *ssooidc.RegisterClientOutput, deviceOutput *ssooidc.StartDeviceAuthorizationOutput) (*ssooidc.CreateTokenOutput, error) {
	interval := time.Duration(deviceOutput.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(deviceOutput.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		if err := sleepContext(ctx, interval); err != nil {
			return nil, err
		}
		output, err := client.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     registerOutput.ClientId,
			ClientSecret: registerOutput.ClientSecret,
			DeviceCode:   deviceOutput.DeviceCode,
			GrantType:    awssdk.String(ssoDeviceCodeGrantType),
		})
		if err == nil {
			return output, nil
		}
		var pending *ssooidctypes.AuthorizationPendingException
		if errors.As(err, &pending) {
			continue
		}
		var slowDown *ssooidctypes.SlowDownException
		if errors.As(err, &slowDown) {
			interval += 5 * time.Second
			continue
		}
		return nil, fmt.Errorf("create SSO token: %w", err)
	}
	return nil, fmt.Errorf("SSO device authorization expired")
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func storeSSOToken(tokenPath string, profile ProfileInfo, registerOutput *ssooidc.RegisterClientOutput, createOutput *ssooidc.CreateTokenOutput) error {
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		return fmt.Errorf("create SSO cache directory: %w", err)
	}
	expiresAt := time.Now().Add(time.Duration(createOutput.ExpiresIn) * time.Second).UTC()
	registrationExpiresAt := time.Unix(registerOutput.ClientSecretExpiresAt, 0).UTC()
	token := ssoCachedToken{
		StartURL:              profile.SSOStartURL,
		Region:                profile.SSORegion,
		AccessToken:           awssdk.ToString(createOutput.AccessToken),
		ExpiresAt:             expiresAt.Format(time.RFC3339),
		RefreshToken:          awssdk.ToString(createOutput.RefreshToken),
		ClientID:              awssdk.ToString(registerOutput.ClientId),
		ClientSecret:          awssdk.ToString(registerOutput.ClientSecret),
		RegistrationExpiresAt: registrationExpiresAt.Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal SSO cache token: %w", err)
	}
	tmpPath := fmt.Sprintf("%s.tmp-%d", tokenPath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write SSO cache token: %w", err)
	}
	if err := os.Rename(tmpPath, tokenPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace SSO cache token: %w", err)
	}
	return nil
}

func openBrowser(uri string) error {
	if uri == "" {
		return nil
	}
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{uri}
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", uri}
	default:
		command = "xdg-open"
		args = []string{uri}
	}
	cmd := exec.CommandContext(context.Background(), command, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
