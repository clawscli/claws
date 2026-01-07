//go:generate go run ../../scripts/gen-imports

package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/app"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

// version is set by ldflags during build
var version = "dev"

func main() {
	opts := parseFlags()

	propagateAllProxy()

	fileCfg := config.File()
	cfg := config.Global()

	if opts.autosave != nil {
		fileCfg.SetPersistenceEnabled(*opts.autosave)
	}

	// Check environment variables (CLI flags take precedence)
	if !opts.readOnly {
		if v := os.Getenv("CLAWS_READ_ONLY"); v == "1" || v == "true" {
			opts.readOnly = true
		}
	}
	cfg.SetReadOnly(opts.readOnly)

	for _, p := range opts.profiles {
		if !config.IsValidProfileName(p) {
			fmt.Fprintf(os.Stderr, "Error: invalid profile name: %s\n", p)
			fmt.Fprintln(os.Stderr, "Valid characters: alphanumeric, hyphen, underscore, period")
			os.Exit(1)
		}
	}
	for _, r := range opts.regions {
		if !config.IsValidRegion(r) {
			fmt.Fprintf(os.Stderr, "Error: invalid region format: %s\n", r)
			fmt.Fprintln(os.Stderr, "Expected: xx-xxxx-N (e.g., us-east-1, ap-northeast-1)")
			os.Exit(1)
		}
	}

	applyStartupConfig(opts, fileCfg, cfg)

	ui.ApplyConfigWithOverride(fileCfg.GetTheme(), opts.theme)

	// Validate and resolve startup service/resource
	var startupPath *app.StartupPath
	if opts.service != "" {
		service, resourceType, err := resolveStartupService(strings.TrimSpace(opts.service))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		startupPath = &app.StartupPath{
			Service:      service,
			ResourceType: resourceType,
			ResourceID:   strings.TrimSpace(opts.resourceID),
		}
	} else if opts.resourceID != "" {
		fmt.Fprintln(os.Stderr, "Error: --resource-id requires --service")
		fmt.Fprintln(os.Stderr, "Example: claws -s ec2 -i i-1234567890abcdef0")
		os.Exit(1)
	}

	// Enable logging if log file specified
	if opts.logFile != "" {
		if err := log.EnableFile(opts.logFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file %s: %v\n", opts.logFile, err)
		} else {
			log.Info("claws started", "profiles", opts.profiles, "regions", opts.regions, "readOnly", opts.readOnly)
		}
	}

	ctx := context.Background()

	application := app.New(ctx, registry.Global, startupPath)

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
	profiles   []string
	regions    []string
	readOnly   bool
	envCreds   bool
	autosave   *bool
	logFile    string
	service    string
	resourceID string
	theme      string
}

// parseFlags parses command line flags and returns options
func parseFlags() cliOptions {
	return parseFlagsFromArgs(os.Args[1:])
}

// parseFlagsFromArgs parses the given args and returns options (testable)
func parseFlagsFromArgs(args []string) cliOptions {
	opts := cliOptions{}
	showHelp := false
	showVersion := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--profile":
			if i+1 < len(args) {
				i++
				for _, p := range strings.Split(args[i], ",") {
					if p = strings.TrimSpace(p); p != "" && !slices.Contains(opts.profiles, p) {
						opts.profiles = append(opts.profiles, p)
					}
				}
			}
		case "-r", "--region":
			if i+1 < len(args) {
				i++
				for _, r := range strings.Split(args[i], ",") {
					if r = strings.TrimSpace(r); r != "" && !slices.Contains(opts.regions, r) {
						opts.regions = append(opts.regions, r)
					}
				}
			}
		case "-ro", "--read-only":
			opts.readOnly = true
		case "-e", "--env":
			opts.envCreds = true
		case "--autosave":
			t := true
			opts.autosave = &t
		case "--no-autosave":
			f := false
			opts.autosave = &f
		case "-l", "--log-file":
			if i+1 < len(args) {
				i++
				opts.logFile = args[i]
			}
		case "-s", "--service":
			if i+1 < len(args) {
				i++
				opts.service = args[i]
			}
		case "-i", "--resource-id":
			if i+1 < len(args) {
				i++
				opts.resourceID = args[i]
			}
		case "-t", "--theme":
			if i+1 < len(args) {
				i++
				opts.theme = args[i]
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
	fmt.Println("  -p, --profile <name>[,name2,...]")
	fmt.Println("        AWS profile(s) to use (comma-separated or repeated)")
	fmt.Println("  -r, --region <region>[,region2,...]")
	fmt.Println("        AWS region(s) to use (comma-separated or repeated)")
	fmt.Println("  -s, --service <service>[/<resource>]")
	fmt.Println("        Start directly on a service/resource (e.g., ec2, rds/snapshots, cfn)")
	fmt.Println("        Special views: dashboard, services")
	fmt.Println("        Supports aliases: cfn, sg, logs, ddb, etc.")
	fmt.Println("  -i, --resource-id <id>")
	fmt.Println("        Open detail view for a specific resource (requires --service)")
	fmt.Println("  -e, --env")
	fmt.Println("        Use environment credentials (ignore ~/.aws config)")
	fmt.Println("        Useful for instance profiles, ECS task roles, Lambda, etc.")
	fmt.Println("  -ro, --read-only")
	fmt.Println("        Run in read-only mode (disable dangerous actions)")
	fmt.Println("  --autosave")
	fmt.Println("        Enable saving region/profile/theme to config file")
	fmt.Println("  --no-autosave")
	fmt.Println("        Disable saving region/profile/theme to config file")
	fmt.Println("  -l, --log-file <path>")
	fmt.Println("        Enable debug logging to specified file")
	fmt.Println("  -t, --theme <name>")
	fmt.Println("        Color theme: dark, light, nord, dracula, gruvbox, catppuccin")
	fmt.Println("  -v, --version")
	fmt.Println("        Show version")
	fmt.Println("  -h, --help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  claws                             Start with service browser (default)")
	fmt.Println("  claws -s dashboard                Start with dashboard")
	fmt.Println("  claws -s services                 Start with service browser")
	fmt.Println("  claws -s ec2                      Open EC2 instances browser")
	fmt.Println("  claws -s rds/snapshots            Open RDS snapshots browser")
	fmt.Println("  claws -s cfn                      Open CloudFormation stacks (alias)")
	fmt.Println("  claws -s ec2 -i i-12345           Open detail view for instance i-12345")
	fmt.Println("  claws -p dev,prod                 Query multiple profiles")
	fmt.Println("  claws -r us-east-1,ap-northeast-1 Query multiple regions")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CLAWS_READ_ONLY=1|true   Enable read-only mode")
	fmt.Println("  ALL_PROXY                Propagated to HTTP_PROXY/HTTPS_PROXY if not set")
}

func applyStartupConfig(opts cliOptions, fileCfg *config.FileConfig, cfg *config.Config) {
	startupRegions, startupProfiles := fileCfg.GetStartup()

	if opts.envCreds {
		cfg.UseEnvOnly()
	} else if len(opts.profiles) > 0 {
		sels := make([]config.ProfileSelection, len(opts.profiles))
		for i, p := range opts.profiles {
			sels[i] = config.ProfileSelectionFromID(p)
		}
		cfg.SetSelections(sels)
	} else if len(startupProfiles) > 0 {
		sels := make([]config.ProfileSelection, len(startupProfiles))
		for i, id := range startupProfiles {
			sels[i] = config.ProfileSelectionFromID(id)
		}
		cfg.SetSelections(sels)
	}

	if len(opts.regions) > 0 {
		cfg.SetRegions(opts.regions)
	} else if len(startupRegions) > 0 {
		cfg.SetRegions(startupRegions)
	}
}

// resolveStartupService validates and resolves a service string (e.g., "ec2", "rds/snapshots", "cfn")
// to a valid service/resourceType pair. Supports aliases and service/resource syntax.
// Special views "dashboard" and "services" are returned as-is.
func resolveStartupService(input string) (service, resourceType string, err error) {
	// Special views: dashboard and services
	if input == "dashboard" || input == "services" {
		return input, "", nil
	}

	return registry.Global.ParseServiceResource(input)
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
