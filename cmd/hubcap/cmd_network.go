package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
)

func cmdConsole(cfg *Config, args []string) int {
	// Parse console-specific flags
	fs := flag.NewFlagSet("console", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	duration := fs.Duration("duration", 0, "How long to capture (0 = until interrupted)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	ctx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

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

	messages, stopCapture, err := client.CaptureConsole(ctx, target.ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	defer stopCapture() // Clean up resources on exit

	enc := json.NewEncoder(cfg.Stdout)
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return ExitSuccess
			}
			if err := enc.Encode(msg); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
				return ExitError
			}
		case <-ctx.Done():
			return ExitSuccess
		}
	}
}

func cmdErrors(cfg *Config, args []string) int {
	// Parse errors-specific flags
	fs := flag.NewFlagSet("errors", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	duration := fs.Duration("duration", 0, "How long to capture (0 = until interrupted)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	ctx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

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

	exceptions, stopCapture, err := client.CaptureExceptions(ctx, target.ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	defer stopCapture()

	enc := json.NewEncoder(cfg.Stdout)
	for {
		select {
		case exc, ok := <-exceptions:
			if !ok {
				return ExitSuccess
			}
			if err := enc.Encode(exc); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
				return ExitError
			}
		case <-ctx.Done():
			return ExitSuccess
		}
	}
}

func cmdNetwork(cfg *Config, args []string) int {
	// Parse network-specific flags
	fs := flag.NewFlagSet("network", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	duration := fs.Duration("duration", 0, "How long to capture (0 = until interrupted)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	ctx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

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

	events, stopCapture, err := client.CaptureNetwork(ctx, target.ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	defer stopCapture() // Clean up resources on exit

	enc := json.NewEncoder(cfg.Stdout)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return ExitSuccess
			}
			if err := enc.Encode(event); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
				return ExitError
			}
		case <-ctx.Done():
			return ExitSuccess
		}
	}
}

func cmdHar(cfg *Config, args []string) int {
	// Parse har-specific flags
	fs := flag.NewFlagSet("har", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	duration := fs.Duration("duration", 5*time.Second, "How long to capture")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.CaptureHAR(ctx, target.ID, *duration)
	})
}

func cmdCoverage(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetCoverage(ctx, target.ID)
	})
}

func cmdStylesheets(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetStylesheets(ctx, target.ID)
	})
}

// InterceptResult is returned by the intercept command.
type InterceptResult struct {
	Enabled     bool   `json:"enabled"`
	Pattern     string `json:"pattern,omitempty"`
	Response    bool   `json:"response,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

func cmdIntercept(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("intercept", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	response := fs.Bool("response", false, "Intercept responses (default: requests)")
	pattern := fs.String("pattern", "*", "URL pattern to match")
	replace := fs.String("replace", "", "Text replacement in format old:new")
	disable := fs.Bool("disable", false, "Disable interception")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	// Disable interception
	if *disable {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.DisableIntercept(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return InterceptResult{Enabled: false}, nil
		})
	}

	// Parse replacement
	replacements := make(map[string]string)
	if *replace != "" {
		parts := strings.SplitN(*replace, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintln(cfg.Stderr, "error: replacement must be in format 'old:new'")
			return ExitError
		}
		replacements[parts[0]] = parts[1]
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		config := chrome.InterceptConfig{
			URLPattern:        *pattern,
			InterceptResponse: *response,
			Replacements:      replacements,
		}
		err := client.EnableIntercept(ctx, target.ID, config)
		if err != nil {
			return nil, err
		}
		return InterceptResult{
			Enabled:     true,
			Pattern:     *pattern,
			Response:    *response,
			Replacement: *replace,
		}, nil
	})
}

// BlockResult is returned by the block command.
type BlockResult struct {
	Enabled  bool     `json:"enabled"`
	Patterns []string `json:"patterns,omitempty"`
}

func cmdBlock(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("block", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	disable := fs.Bool("disable", false, "Disable URL blocking")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	// Disable blocking
	if *disable {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.UnblockURLs(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return BlockResult{Enabled: false}, nil
		})
	}

	// Get URL patterns from remaining args
	patterns := fs.Args()
	if len(patterns) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap block <pattern>... [--disable]")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.BlockURLs(ctx, target.ID, patterns)
		if err != nil {
			return nil, err
		}
		return BlockResult{
			Enabled:  true,
			Patterns: patterns,
		}, nil
	})
}

func cmdResponseBody(cfg *Config, requestID string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetResponseBody(ctx, target.ID, requestID)
	})
}

func cmdCSSCoverage(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetCSSCoverage(ctx, target.ID)
	})
}

func cmdDOMSnapshot(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetDOMSnapshot(ctx, target.ID)
	})
}

// ThrottleResult is returned by the throttle command.
type ThrottleResult struct {
	Preset  string `json:"preset,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

// ThrottleDisabledResult is returned when throttling is disabled.
type ThrottleDisabledResult struct {
	Disabled bool `json:"disabled"`
}

func cmdThrottle(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("throttle", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	disable := fs.Bool("disable", false, "Disable network throttling")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()

	if *disable {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.DisableNetworkThrottling(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return ThrottleDisabledResult{Disabled: true}, nil
		})
	}

	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap throttle <preset> | --disable")
		fmt.Fprintln(cfg.Stderr, "\nAvailable presets:")
		for name := range chrome.NetworkPresets {
			fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
		}
		return ExitError
	}

	preset := remaining[0]
	conditions, ok := chrome.NetworkPresets[preset]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: unknown preset %q\n", preset)
		fmt.Fprintln(cfg.Stderr, "\nAvailable presets:")
		for name := range chrome.NetworkPresets {
			fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.EmulateNetworkConditions(ctx, target.ID, conditions)
		if err != nil {
			return nil, err
		}
		return ThrottleResult{Preset: preset, Enabled: true}, nil
	})
}
