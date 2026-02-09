package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
)

// Exit codes
const (
	ExitSuccess    = 0
	ExitError      = 1
	ExitConnFailed = 2
	ExitTimeout    = 3
)

// Config holds the CLI configuration.
type Config struct {
	Port    int
	Host    string
	Timeout time.Duration
	Output  string // json, ndjson, text
	Quiet   bool
	Target  string // target index or ID

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// PortChecker overrides port detection for testing. If nil, uses launcher.IsPortOpen.
	PortChecker func(host string, port int) bool
}

// DefaultConfig returns the default configuration with built-in defaults.
// Environment variables and profiles are applied later in the config chain.
func DefaultConfig() *Config {
	return &Config{
		Port:    9222,
		Host:    "localhost",
		Timeout: 10 * time.Second,
		Output:  "json",
		Quiet:   false,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

func main() {
	cfg := DefaultConfig()
	os.Exit(run(os.Args[1:], cfg))
}

// flagValues stores values parsed from CLI flags before they get overwritten.
type flagValues struct {
	port    int
	host    string
	timeout time.Duration
	output  string
	quiet   bool
	target  string
}

func run(args []string, cfg *Config) int {
	// Parse into temporary variables so we can snapshot values before overwriting
	var fv flagValues
	fs := flag.NewFlagSet("hubcap", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	fs.IntVar(&fv.port, "port", cfg.Port, "Chrome debug port (env: HUBCAP_PORT)")
	fs.StringVar(&fv.host, "host", cfg.Host, "Chrome debug host (env: HUBCAP_HOST)")
	fs.DurationVar(&fv.timeout, "timeout", cfg.Timeout, "Command timeout")
	fs.StringVar(&fv.output, "output", cfg.Output, "Output format: json, ndjson, text")
	fs.BoolVar(&fv.quiet, "quiet", cfg.Quiet, "Suppress non-essential output")
	fs.StringVar(&fv.target, "target", cfg.Target, "Target page (index or ID)")
	profileName := fs.String("profile", "", "Named profile (env: HUBCAP_PROFILE)")
	helpCommands := fs.Bool("help-commands", false, "List all commands with descriptions")

	fs.Usage = func() { printBriefUsage(cfg, fs) }

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	// Track which flags were explicitly set on the command line
	explicitFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	// Clean up stale ephemeral sessions
	cleanupStaleEphemeral(configDir())

	// Config precedence: built-in defaults < profile < .hubcaprc < env vars < CLI flags
	// 1. Apply profile (if any)
	if code := applyProfile(cfg, *profileName); code != -1 {
		return code
	}

	// 2. Apply .hubcaprc
	loadConfigFile(cfg)

	// 3. Apply env vars (only if not explicitly set by flags)
	applyEnvVars(cfg, explicitFlags)

	// 4. Re-apply explicit CLI flags on top of everything
	reapplyExplicitFlags(cfg, &fv, explicitFlags)

	if *helpCommands {
		printFullCommandList(cfg)
		return ExitSuccess
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		printBriefUsage(cfg, fs)
		return ExitError
	}

	cmd := remaining[0]

	info, ok := commands[cmd]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", cmd)
		return ExitError
	}
	return info.Run(cfg, remaining[1:])
}

// applyProfile loads and applies the named profile to cfg.
// Returns -1 if successful, or an exit code on error.
func applyProfile(cfg *Config, flagProfile string) int {
	// Resolve profile name: --profile flag > HUBCAP_PROFILE env > default from file
	name := flagProfile
	if name == "" {
		name = os.Getenv("HUBCAP_PROFILE")
	}

	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	if name == "" {
		name = pf.Default
	}

	// No profile to apply
	if name == "" {
		return -1
	}

	p, ok := pf.Profiles[name]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	// Apply profile fields (only non-zero values)
	if p.Host != "" {
		cfg.Host = p.Host
	}
	if p.Port != 0 {
		cfg.Port = p.Port
	}
	if p.Timeout != "" {
		if d, err := time.ParseDuration(p.Timeout); err == nil {
			cfg.Timeout = d
		}
	}
	if p.Output != "" {
		cfg.Output = p.Output
	}
	if p.Target != "" {
		cfg.Target = p.Target
	}

	// Ephemeral profile: auto-launch Chrome if needed
	if p.Ephemeral {
		port, err := ensureEphemeralRunning(dir, name, p)
		if err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
		cfg.Port = port
	}

	return -1
}

// applyEnvVars applies environment variables to cfg, but only for fields
// not already set by explicit CLI flags.
func applyEnvVars(cfg *Config, explicit map[string]bool) {
	if !explicit["port"] {
		if v := os.Getenv("HUBCAP_PORT"); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				cfg.Port = i
			}
		}
	}
	if !explicit["host"] {
		if v := os.Getenv("HUBCAP_HOST"); v != "" {
			cfg.Host = v
		}
	}
}

// reapplyExplicitFlags re-applies flag values that were explicitly set
// on the command line, since profile/.hubcaprc loading may have overwritten them.
func reapplyExplicitFlags(cfg *Config, fv *flagValues, explicit map[string]bool) {
	if explicit["port"] {
		cfg.Port = fv.port
	}
	if explicit["host"] {
		cfg.Host = fv.host
	}
	if explicit["timeout"] {
		cfg.Timeout = fv.timeout
	}
	if explicit["output"] {
		cfg.Output = fv.output
	}
	if explicit["quiet"] {
		cfg.Quiet = fv.quiet
	}
	if explicit["target"] {
		cfg.Target = fv.target
	}
}

// resolveTarget resolves the target page from cfg.Target.
// If cfg.Target is empty, returns the first page.
// If cfg.Target is a number, uses it as an index into the pages list.
// Otherwise, treats cfg.Target as a target ID.
func resolveTarget(ctx context.Context, client *chrome.Client, cfg *Config) (*chrome.TargetInfo, error) {
	pages, err := client.Pages(ctx)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages available")
	}

	// Default: first page
	if cfg.Target == "" {
		return &pages[0], nil
	}

	// Try as index first
	if idx, err := strconv.Atoi(cfg.Target); err == nil {
		if idx < 0 || idx >= len(pages) {
			return nil, fmt.Errorf("invalid target index: %d (have %d pages)", idx, len(pages))
		}
		return &pages[idx], nil
	}

	// Otherwise, treat as target ID
	for i := range pages {
		if pages[i].ID == cfg.Target {
			return &pages[i], nil
		}
	}

	return nil, fmt.Errorf("invalid target: %s (not found)", cfg.Target)
}

// withClient executes a function with a connected Chrome client.
func withClient(cfg *Config, fn func(ctx context.Context, client *chrome.Client) (interface{}, error)) int {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := chrome.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	result, err := fn(ctx, client)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintln(cfg.Stderr, "error: timeout")
			return ExitTimeout
		}
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, result)
}

// withClientTarget executes a function with a connected Chrome client and resolved target.
func withClientTarget(cfg *Config, fn func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error)) int {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := chrome.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	target, err := resolveTarget(ctx, client, cfg)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	result, err := fn(ctx, client, target)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintln(cfg.Stderr, "error: timeout")
			return ExitTimeout
		}
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, result)
}
