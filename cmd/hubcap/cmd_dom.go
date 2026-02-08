package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/tomyan/hubcap/internal/chrome"
)

func cmdQuery(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.Query(ctx, target.ID, selector)
	})
}

// HTMLResult is returned by the html command.
type HTMLResult struct {
	Selector string `json:"selector"`
	HTML     string `json:"html"`
}

func cmdHTML(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		html, err := client.GetHTML(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return HTMLResult{Selector: selector, HTML: html}, nil
	})
}

// TextResult is returned by the text command.
type TextResult struct {
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

func cmdText(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		text, err := client.GetText(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return TextResult{Selector: selector, Text: text}, nil
	})
}

// AttrResult is returned by the attr command.
type AttrResult struct {
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

func cmdAttr(cfg *Config, selector, attribute string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		value, err := client.GetAttribute(ctx, target.ID, selector, attribute)
		if err != nil {
			return nil, err
		}
		return AttrResult{Selector: selector, Attribute: attribute, Value: value}, nil
	})
}

// ValueResult is returned by the value command.
type ValueResult struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

func cmdValue(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		value, err := client.GetValue(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ValueResult{Selector: selector, Value: value}, nil
	})
}

// CountResult is returned by the count command.
type CountResult struct {
	Count    int    `json:"count"`
	Selector string `json:"selector"`
}

func cmdCount(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		count, err := client.CountElements(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return CountResult{Count: count, Selector: selector}, nil
	})
}

// VisibleResult is returned by the visible command.
type VisibleResult struct {
	Visible  bool   `json:"visible"`
	Selector string `json:"selector"`
}

func cmdVisible(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		visible, err := client.IsVisible(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return VisibleResult{Visible: visible, Selector: selector}, nil
	})
}

// ExistsResult is returned by the exists command.
type ExistsResult struct {
	Exists   bool   `json:"exists"`
	Selector string `json:"selector"`
}

func cmdExists(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		exists, err := client.Exists(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ExistsResult{Exists: exists, Selector: selector}, nil
	})
}

func cmdBounds(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		bounds, err := client.GetBoundingBox(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return bounds, nil
	})
}

type StylesResult struct {
	Selector string            `json:"selector"`
	Styles   map[string]string `json:"styles"`
}

func cmdStyles(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		styles, err := client.GetComputedStyles(ctx, target.ID, selector, nil)
		if err != nil {
			return nil, err
		}
		return StylesResult{Selector: selector, Styles: styles}, nil
	})
}

func cmdComputed(cfg *Config, selector, property string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.GetComputedStyle(ctx, target.ID, selector, property)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func cmdLayout(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("layout", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	depth := fs.Int("depth", 1, "Depth of children to include (0=element only, 1=immediate children)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap layout <selector> [--depth <n>]")
		return ExitError
	}

	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		layout, err := client.GetElementLayout(ctx, target.ID, selector, *depth)
		if err != nil {
			return nil, err
		}
		return layout, nil
	})
}

// ShadowResult is returned by the shadow command.
type ShadowResult struct {
	HostSelector  string            `json:"hostSelector"`
	InnerSelector string            `json:"innerSelector"`
	NodeID        int               `json:"nodeId"`
	TagName       string            `json:"tagName"`
	Attributes    map[string]string `json:"attributes"`
}

func cmdShadow(cfg *Config, hostSelector, innerSelector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.QueryShadow(ctx, target.ID, hostSelector, innerSelector)
		if err != nil {
			return nil, err
		}
		return ShadowResult{
			HostSelector:  hostSelector,
			InnerSelector: innerSelector,
			NodeID:        result.NodeID,
			TagName:       result.TagName,
			Attributes:    result.Attributes,
		}, nil
	})
}

func cmdFind(cfg *Config, text string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.FindText(ctx, target.ID, text)
	})
}

func cmdSelection(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.GetSelection(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func cmdCaret(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.GetCaretPosition(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func cmdSetValue(cfg *Config, selector string, value string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.SetValue(ctx, target.ID, selector, value)
	})
}

func cmdListeners(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetEventListeners(ctx, target.ID, selector)
	})
}

// HighlightResult is returned by the highlight command.
type HighlightResult struct {
	Highlighted bool   `json:"highlighted"`
	Selector    string `json:"selector,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
}

func cmdHighlight(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("highlight", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	hide := fs.Bool("hide", false, "Hide existing highlight")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *hide {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.HideHighlight(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return HighlightResult{Hidden: true}, nil
		})
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap highlight <selector> [--hide]")
		return ExitError
	}
	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Highlight(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return HighlightResult{Highlighted: true, Selector: selector}, nil
	})
}
