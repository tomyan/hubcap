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

	Stdout io.Writer
	Stderr io.Writer
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:    getEnvInt("HUBCAP_PORT", 9222),
		Host:    getEnv("HUBCAP_HOST", "localhost"),
		Timeout: 10 * time.Second,
		Output:  "json",
		Quiet:   false,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func main() {
	cfg := DefaultConfig()
	loadConfigFile(cfg)
	os.Exit(run(os.Args[1:], cfg))
}

func run(args []string, cfg *Config) int {
	// Parse global flags
	fs := flag.NewFlagSet("hubcap", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "Chrome debug port (env: HUBCAP_PORT)")
	fs.StringVar(&cfg.Host, "host", cfg.Host, "Chrome debug host (env: HUBCAP_HOST)")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Command timeout")
	fs.StringVar(&cfg.Output, "output", cfg.Output, "Output format: json, ndjson, text")
	fs.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "Suppress non-essential output")
	fs.StringVar(&cfg.Target, "target", cfg.Target, "Target page (index or ID)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		printUsage(cfg, fs)
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
