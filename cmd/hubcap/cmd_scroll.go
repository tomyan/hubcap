package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tomyan/hubcap/internal/chrome"
)

// ScrollToResult is returned by the scrollto command.
type ScrollToResult struct {
	Scrolled bool   `json:"scrolled"`
	Selector string `json:"selector"`
}

func cmdScrollTo(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.ScrollIntoView(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ScrollToResult{Scrolled: true, Selector: selector}, nil
	})
}

// ScrollResult is returned by the scroll command.
type ScrollResult struct {
	Scrolled bool `json:"scrolled"`
	X        int  `json:"x"`
	Y        int  `json:"y"`
}

func cmdScroll(cfg *Config, xStr, yStr string) int {
	x, err := strconv.Atoi(xStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid x value: %s\n", xStr)
		return ExitError
	}
	y, err := strconv.Atoi(yStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid y value: %s\n", yStr)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.ScrollBy(ctx, target.ID, x, y)
		if err != nil {
			return nil, err
		}
		return ScrollResult{Scrolled: true, X: x, Y: y}, nil
	})
}

// ScrollBottomResult is returned by the scrollbottom command.
type ScrollBottomResult struct {
	Scrolled bool `json:"scrolled"`
}

func cmdScrollBottom(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.ScrollToBottom(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ScrollBottomResult{Scrolled: true}, nil
	})
}

// ScrollTopResult is returned by the scrolltop command.
type ScrollTopResult struct {
	Scrolled bool `json:"scrolled"`
}

func cmdScrollTop(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.ScrollToTop(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ScrollTopResult{Scrolled: true}, nil
	})
}

// ViewportResult is returned by the viewport command.
type ViewportResult struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func cmdViewport(cfg *Config, widthStr, heightStr string) int {
	width, err := strconv.Atoi(widthStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid width: %s\n", widthStr)
		return ExitError
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid height: %s\n", heightStr)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetViewport(ctx, target.ID, width, height)
		if err != nil {
			return nil, err
		}
		return ViewportResult{Width: width, Height: height}, nil
	})
}
