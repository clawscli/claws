//go:generate go run ../../scripts/gen-imports

package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/app"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
)

// version is set by ldflags during build
var version = "dev"

func main() {
	opts := parseFlags()

	propagateAllProxy()

	fileCfg := config.File()
	cfg := config.Global()

	// CLI persistence flags override config file
	if opts.persist != nil {
		fileCfg.SetPersistenceEnabled(*opts.persist)
	}

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

	applyStartupConfig(opts, fileCfg, cfg)

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

type cliOptions struct {
	profile  string
	region   string
	readOnly bool
	envCreds bool
	persist  *bool // nil = use config, true = enable, false = disable
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
		case "--persist":
			t := true
			opts.persist = &t
		case "--no-persist":
			f := false
			opts.persist = &f
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
	fmt.Println("  --persist")
	fmt.Println("        Enable saving region/profile selection to config file")
	fmt.Println("  --no-persist")
	fmt.Println("        Disable saving region/profile selection to config file")
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
}

// applyStartupConfig applies profile/region config with precedence:
// 1. CLI flags (-p, -r, -e) - highest priority
// 2. Config file startup section
// 3. AWS SDK defaults
func applyStartupConfig(opts cliOptions, fileCfg *config.FileConfig, cfg *config.Config) {
	startupRegions, startupProfile := fileCfg.GetStartup()

	// Apply profile: CLI > startup config
	if opts.envCreds {
		cfg.UseEnvOnly()
	} else if opts.profile != "" {
		cfg.UseProfile(opts.profile)
	} else if startupProfile != "" {
		cfg.UseProfile(startupProfile)
	}

	// Apply region: CLI > startup config
	if opts.region != "" {
		cfg.SetRegion(opts.region)
	} else if len(startupRegions) > 0 {
		cfg.SetRegions(startupRegions)
	}
}

// propagateAllProxy copies ALL_PROXY to HTTP_PROXY/HTTPS_PROXY if not set.
// Go's net/http ignores ALL_PROXY, so we propagate it to the standard vars.
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
