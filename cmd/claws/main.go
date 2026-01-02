//go:generate go run ../../scripts/gen-imports

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/app"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
)

// version is set by ldflags during build
var version = "dev"

func main() {
	propagateAllProxy()
	configureNoProxy()

	opts := parseFlags()

	// Apply CLI options to global config
	cfg := config.Global()

	// Check environment variables (CLI flags take precedence)
	if !opts.readOnly {
		if v := os.Getenv("CLAWS_READ_ONLY"); v == "1" || v == "true" {
			opts.readOnly = true
		}
	}
	cfg.SetReadOnly(opts.readOnly)

	if opts.profile != "" && !config.IsValidProfileName(opts.profile) {
		fmt.Fprintf(os.Stderr, "Error: invalid profile name: %s\n", opts.profile)
		fmt.Fprintln(os.Stderr, "Valid characters: alphanumeric, hyphen, underscore, period")
		os.Exit(1)
	}
	if opts.region != "" && !config.IsValidRegion(opts.region) {
		fmt.Fprintf(os.Stderr, "Error: invalid region format: %s\n", opts.region)
		fmt.Fprintln(os.Stderr, "Expected: xx-xxxx-N (e.g., us-east-1, ap-northeast-1)")
		os.Exit(1)
	}

	if opts.envCreds {
		// Use environment credentials, ignore ~/.aws config
		cfg.UseEnvOnly()
	} else if opts.profile != "" {
		cfg.UseProfile(opts.profile)
		// Don't set AWS_PROFILE globally - it interferes with EnvOnly mode
		// when switching profiles. SelectionLoadOptions uses WithSharedConfigProfile
		// for SDK calls, and BuildSubprocessEnv handles subprocess environment.
	}
	// else: SDKDefault is the zero value, no action needed
	if opts.region != "" {
		cfg.SetRegion(opts.region)
		// Don't set AWS_REGION globally - SelectionLoadOptions handles SDK calls,
		// and BuildSubprocessEnv handles subprocess environment.
	}

	// Enable logging if log file specified
	if opts.logFile != "" {
		if err := log.EnableFile(opts.logFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file %s: %v\n", opts.logFile, err)
		} else {
			log.Info("claws started", "profile", opts.profile, "region", opts.region, "readOnly", opts.readOnly)
		}
	}

	ctx := context.Background()

	// Create the application
	application := app.New(ctx, registry.Global)

	// Run the TUI
	// Note: In v2, AltScreen and MouseMode are set via the View struct
	// v2 has better ESC key handling via x/input package
	p := tea.NewProgram(application)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cliOptions holds command line options
type cliOptions struct {
	profile  string
	region   string
	readOnly bool
	envCreds bool
	logFile  string
}

// parseFlags parses command line flags and returns options
func parseFlags() cliOptions {
	opts := cliOptions{}
	showHelp := false
	showVersion := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--profile":
			if i+1 < len(args) {
				i++
				opts.profile = args[i]
			}
		case "-r", "--region":
			if i+1 < len(args) {
				i++
				opts.region = args[i]
			}
		case "-ro", "--read-only":
			opts.readOnly = true
		case "-e", "--env":
			opts.envCreds = true
		case "-l", "--log-file":
			if i+1 < len(args) {
				i++
				opts.logFile = args[i]
			}
		case "-h", "--help":
			showHelp = true
		case "-v", "--version":
			showVersion = true
		}
	}

	if showVersion {
		fmt.Printf("claws %s\n", version)
		os.Exit(0)
	}

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	return opts
}

func printUsage() {
	fmt.Println("claws - A terminal UI for AWS resource management")
	fmt.Println()
	fmt.Println("Usage: claws [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -p, --profile <name>")
	fmt.Println("        AWS profile to use")
	fmt.Println("  -r, --region <region>")
	fmt.Println("        AWS region to use")
	fmt.Println("  -e, --env")
	fmt.Println("        Use environment credentials (ignore ~/.aws config)")
	fmt.Println("        Useful for instance profiles, ECS task roles, Lambda, etc.")
	fmt.Println("  -ro, --read-only")
	fmt.Println("        Run in read-only mode (disable dangerous actions)")
	fmt.Println("  -l, --log-file <path>")
	fmt.Println("        Enable debug logging to specified file")
	fmt.Println("  -v, --version")
	fmt.Println("        Show version")
	fmt.Println("  -h, --help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CLAWS_READ_ONLY=1|true   Enable read-only mode")
	fmt.Println("  ALL_PROXY                Propagated to HTTP_PROXY/HTTPS_PROXY if not set")
	fmt.Println("                           NO_PROXY auto-configured for EC2 IMDS (169.254.169.254)")
}

// awsNoProxyEndpoints lists AWS credential endpoints that must bypass proxy.
// Default: EC2 IMDS only. ECS/EKS endpoints can be added via config.
// TODO(#67): make configurable via config.yaml
var awsNoProxyEndpoints = []string{
	"169.254.169.254", // EC2 IMDS
}

// propagateAllProxy copies ALL_PROXY to HTTP_PROXY/HTTPS_PROXY if not set.
// Go's net/http ignores ALL_PROXY. When HTTP_PROXY is set, NO_PROXY is
// configured to exclude AWS credential endpoints (IMDS, ECS, EKS).
func propagateAllProxy() {
	allProxy := os.Getenv("ALL_PROXY")
	if allProxy == "" {
		return
	}

	var propagated []string

	if os.Getenv("HTTPS_PROXY") == "" {
		if err := os.Setenv("HTTPS_PROXY", allProxy); err != nil {
			log.Warn("failed to set HTTPS_PROXY", "error", err)
		} else {
			propagated = append(propagated, "HTTPS_PROXY")
		}
	}

	if os.Getenv("HTTP_PROXY") == "" {
		if err := os.Setenv("HTTP_PROXY", allProxy); err != nil {
			log.Warn("failed to set HTTP_PROXY", "error", err)
		} else {
			propagated = append(propagated, "HTTP_PROXY")
		}
	}

	if len(propagated) > 0 {
		log.Debug("propagated ALL_PROXY", "to", propagated)
	}
}

// configureNoProxy appends awsNoProxyEndpoints to NO_PROXY if missing.
func configureNoProxy() {
	if os.Getenv("HTTP_PROXY") == "" && os.Getenv("HTTPS_PROXY") == "" {
		return
	}

	existing := os.Getenv("NO_PROXY")

	existingSet := make(map[string]bool)
	if existing != "" {
		for _, entry := range splitNoProxy(existing) {
			existingSet[entry] = true
		}
	}

	var additions []string
	for _, endpoint := range awsNoProxyEndpoints {
		if !existingSet[endpoint] {
			additions = append(additions, endpoint)
		}
	}

	if len(additions) == 0 {
		return
	}

	newValue := existing
	if newValue != "" {
		newValue += ","
	}
	for i, endpoint := range additions {
		if i > 0 {
			newValue += ","
		}
		newValue += endpoint
	}

	if err := os.Setenv("NO_PROXY", newValue); err != nil {
		log.Warn("failed to set NO_PROXY", "error", err)
		return
	}
	log.Debug("configured NO_PROXY for AWS endpoints", "added", additions)
}

func splitNoProxy(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			result = append(result, s)
		}
	}
	return result
}
