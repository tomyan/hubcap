package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/tomyan/hubcap/internal/chrome"
)

func cmdVersion(cfg *Config) int {
	return withClient(cfg, func(ctx context.Context, client *chrome.Client) (interface{}, error) {
		return client.Version(ctx)
	})
}

func cmdTabs(cfg *Config) int {
	return withClient(cfg, func(ctx context.Context, client *chrome.Client) (interface{}, error) {
		return client.Pages(ctx)
	})
}

// GotoWaitResult is returned by the goto command with --wait.
type GotoWaitResult struct {
	URL      string `json:"url"`
	FrameID  string `json:"frameId,omitempty"`
	LoaderID string `json:"loaderId,omitempty"`
	Loaded   bool   `json:"loaded"`
}

func cmdGoto(cfg *Config, args []string) int {
	// Parse goto-specific flags
	fs := flag.NewFlagSet("goto", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	wait := fs.Bool("wait", false, "Wait for page load to complete")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap goto [--wait] <url>")
		return ExitError
	}

	url := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		if *wait {
			result, err := client.NavigateAndWait(ctx, target.ID, url)
			if err != nil {
				return nil, err
			}
			return GotoWaitResult{
				URL:      url,
				FrameID:  result.FrameID,
				LoaderID: result.LoaderID,
				Loaded:   true,
			}, nil
		}
		return client.Navigate(ctx, target.ID, url)
	})
}

// ReloadResult is returned by the reload command.
type ReloadResult struct {
	Reloaded    bool `json:"reloaded"`
	IgnoreCache bool `json:"ignoreCache"`
}

func cmdReload(cfg *Config, args []string) int {
	// Parse reload-specific flags
	fs := flag.NewFlagSet("reload", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	ignoreCache := fs.Bool("bypass-cache", false, "Bypass browser cache")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Reload(ctx, target.ID, *ignoreCache)
		if err != nil {
			return nil, err
		}
		return ReloadResult{Reloaded: true, IgnoreCache: *ignoreCache}, nil
	})
}

// BackResult is returned by the back command.
type BackResult struct {
	Success bool `json:"success"`
}

func cmdBack(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.GoBack(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return BackResult{Success: true}, nil
	})
}

// ForwardResult is returned by the forward command.
type ForwardResult struct {
	Success bool `json:"success"`
}

func cmdForward(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.GoForward(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ForwardResult{Success: true}, nil
	})
}

// TitleResult is returned by the title command.
type TitleResult struct {
	Title string `json:"title"`
}

func cmdTitle(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		title, err := client.GetTitle(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return TitleResult{Title: title}, nil
	})
}

// URLResult is returned by the url command.
type URLResult struct {
	URL string `json:"url"`
}

func cmdURL(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		url, err := client.GetURL(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return URLResult{URL: url}, nil
	})
}

// NewTabResult is returned by the new command.
type NewTabResult struct {
	TargetID string `json:"targetId"`
	URL      string `json:"url"`
}

func cmdNew(cfg *Config, url string) int {
	return withClient(cfg, func(ctx context.Context, client *chrome.Client) (interface{}, error) {
		targetID, err := client.NewTab(ctx, url)
		if err != nil {
			return nil, err
		}
		if url == "" {
			url = "about:blank"
		}
		return NewTabResult{TargetID: targetID, URL: url}, nil
	})
}

// CloseTabResult is returned by the close command.
type CloseTabResult struct {
	Closed   bool   `json:"closed"`
	TargetID string `json:"targetId"`
}

func cmdClose(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.CloseTab(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return CloseTabResult{Closed: true, TargetID: target.ID}, nil
	})
}
