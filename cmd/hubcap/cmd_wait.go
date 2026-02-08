package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
)

// WaitResult is returned by the wait command.
type WaitResult struct {
	Found    bool   `json:"found"`
	Selector string `json:"selector"`
}

func cmdWait(cfg *Config, args []string) int {
	// Parse wait-specific flags
	fs := flag.NewFlagSet("wait", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap wait <selector> [--timeout <duration>]")
		return ExitError
	}
	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.WaitFor(ctx, target.ID, selector, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitResult{Found: true, Selector: selector}, nil
	})
}

// WaitTextResult is returned by the waittext command.
type WaitTextResult struct {
	Text  string `json:"text"`
	Found bool   `json:"found"`
}

func cmdWaitText(cfg *Config, text string, args []string) int {
	// Parse waittext-specific flags
	fs := flag.NewFlagSet("waittext", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.WaitForText(ctx, target.ID, text, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitTextResult{Text: text, Found: true}, nil
	})
}

// WaitGoneResult is returned by the waitgone command.
type WaitGoneResult struct {
	Gone     bool   `json:"gone"`
	Selector string `json:"selector"`
}

func cmdWaitGone(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("waitgone", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap waitgone <selector> [--timeout <duration>]")
		return ExitError
	}
	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.WaitForGone(ctx, target.ID, selector, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitGoneResult{Gone: true, Selector: selector}, nil
	})
}

// WaitFnResult is returned by the waitfn command.
type WaitFnResult struct {
	Completed  bool   `json:"completed"`
	Expression string `json:"expression"`
}

func cmdWaitFn(cfg *Config, args []string) int {
	// Parse waitfn-specific flags
	fs := flag.NewFlagSet("waitfn", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap waitfn <expression> [--timeout <duration>]")
		return ExitError
	}
	expression := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.WaitForFunction(ctx, target.ID, expression, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitFnResult{Completed: true, Expression: expression}, nil
	})
}

// WaitIdleResult is returned by the waitidle command.
type WaitIdleResult struct {
	Idle bool `json:"idle"`
}

func cmdWaitIdle(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("waitidle", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	idleTime := fs.Duration("idle", 500*time.Millisecond, "Time with no network activity to consider idle")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.WaitForNetworkIdle(ctx, target.ID, *idleTime)
		if err != nil {
			return nil, err
		}
		return WaitIdleResult{Idle: true}, nil
	})
}

// WaitNavResult is returned by the waitnav command.
type WaitNavResult struct {
	Navigated bool `json:"navigated"`
}

func cmdWaitNav(cfg *Config, args []string) int {
	// Parse waitnav-specific flags
	fs := flag.NewFlagSet("waitnav", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.WaitForNavigation(ctx, target.ID, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitNavResult{Navigated: true}, nil
	})
}

// WaitLoadResult is returned by the waitload command.
type WaitLoadResult struct {
	Loaded bool `json:"loaded"`
}

func cmdWaitLoad(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("waitload", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
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

	err = client.WaitForLoad(ctx, target.ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, WaitLoadResult{Loaded: true})
}

// WaitURLResult is returned by the waiturl command.
type WaitURLResult struct {
	Pattern string `json:"pattern"`
	URL     string `json:"url"`
}

func cmdWaitURL(cfg *Config, pattern string, args []string) int {
	// Parse waiturl-specific flags
	fs := flag.NewFlagSet("waiturl", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		url, err := client.WaitForURL(ctx, target.ID, pattern, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitURLResult{Pattern: pattern, URL: url}, nil
	})
}

func cmdWaitRequest(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("waitrequest", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap waitrequest <pattern> [--timeout <duration>]")
		return ExitError
	}
	pattern := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.WaitForRequest(ctx, target.ID, pattern, *timeout)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func cmdWaitResponse(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("waitresponse", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	timeout := fs.Duration("timeout", 30*time.Second, "Max wait time")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap waitresponse <pattern> [--timeout <duration>]")
		return ExitError
	}
	pattern := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.WaitForResponse(ctx, target.ID, pattern, *timeout)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}
