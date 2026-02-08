package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tomyan/hubcap/internal/chrome"
)

// ClickResult is returned by the click command.
type ClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdClick(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Click(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ClickResult{Clicked: true, Selector: selector}, nil
	})
}

// ClickAtResult is returned by the clickat command.
type ClickAtResult struct {
	Clicked bool    `json:"clicked"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

func cmdClickAt(cfg *Config, xStr, yStr string) int {
	x, err := strconv.ParseFloat(xStr, 64)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: invalid x coordinate: %v\n", err)
		return ExitError
	}
	y, err := strconv.ParseFloat(yStr, 64)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: invalid y coordinate: %v\n", err)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.ClickAt(ctx, target.ID, x, y)
		if err != nil {
			return nil, err
		}
		return ClickAtResult{Clicked: true, X: x, Y: y}, nil
	})
}

// FillResult is returned by the fill command.
type FillResult struct {
	Filled   bool   `json:"filled"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

func cmdFill(cfg *Config, selector, text string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Fill(ctx, target.ID, selector, text)
		if err != nil {
			return nil, err
		}
		return FillResult{Filled: true, Selector: selector, Text: text}, nil
	})
}

// TypeResult is returned by the type command.
type TypeResult struct {
	Typed bool   `json:"typed"`
	Text  string `json:"text"`
}

func cmdType(cfg *Config, text string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Type(ctx, target.ID, text)
		if err != nil {
			return nil, err
		}
		return TypeResult{Typed: true, Text: text}, nil
	})
}

// FocusResult is returned by the focus command.
type FocusResult struct {
	Focused  bool   `json:"focused"`
	Selector string `json:"selector"`
}

func cmdFocus(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Focus(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return FocusResult{Focused: true, Selector: selector}, nil
	})
}

// PressResult is returned by the press command.
type PressResult struct {
	Pressed bool   `json:"pressed"`
	Key     string `json:"key"`
}

func cmdPress(cfg *Config, key string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		// Parse modifier+key combinations like "Ctrl+A", "Shift+End", "Ctrl+Shift+N"
		mods := chrome.KeyModifiers{}
		actualKey := key

		parts := strings.Split(key, "+")
		if len(parts) > 1 {
			actualKey = parts[len(parts)-1]
			for _, mod := range parts[:len(parts)-1] {
				switch strings.ToLower(mod) {
				case "ctrl", "control":
					mods.Ctrl = true
				case "alt":
					mods.Alt = true
				case "shift":
					mods.Shift = true
				case "meta", "cmd", "command":
					mods.Meta = true
				}
			}
		}

		err := client.PressKeyWithModifiers(ctx, target.ID, actualKey, mods)
		if err != nil {
			return nil, err
		}
		return PressResult{Pressed: true, Key: key}, nil
	})
}

// HoverResult is returned by the hover command.
type HoverResult struct {
	Hovered  bool   `json:"hovered"`
	Selector string `json:"selector"`
}

func cmdHover(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Hover(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return HoverResult{Hovered: true, Selector: selector}, nil
	})
}

// TapResult is returned by the tap command.
type TapResult struct {
	Tapped   bool   `json:"tapped"`
	Selector string `json:"selector"`
}

func cmdTap(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Tap(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return TapResult{Tapped: true, Selector: selector}, nil
	})
}

// DragResult is returned by the drag command.
type DragResult struct {
	Dragged bool   `json:"dragged"`
	Source  string `json:"source"`
	Dest    string `json:"dest"`
}

func cmdDrag(cfg *Config, source, dest string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Drag(ctx, target.ID, source, dest)
		if err != nil {
			return nil, err
		}
		return DragResult{Dragged: true, Source: source, Dest: dest}, nil
	})
}

// DblClickResult is returned by the dblclick command.
type DblClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdDblClick(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.DoubleClick(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return DblClickResult{Clicked: true, Selector: selector}, nil
	})
}

// RightClickResult is returned by the rightclick command.
type RightClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdRightClick(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.RightClick(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return RightClickResult{Clicked: true, Selector: selector}, nil
	})
}

// ClearResult is returned by the clear command.
type ClearResult struct {
	Cleared  bool   `json:"cleared"`
	Selector string `json:"selector"`
}

func cmdClear(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Clear(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ClearResult{Cleared: true, Selector: selector}, nil
	})
}

// SelectResult is returned by the select command.
type SelectResult struct {
	Selected bool   `json:"selected"`
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

func cmdSelect(cfg *Config, selector, value string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SelectOption(ctx, target.ID, selector, value)
		if err != nil {
			return nil, err
		}
		return SelectResult{Selected: true, Selector: selector, Value: value}, nil
	})
}

// CheckResult is returned by the check command.
type CheckResult struct {
	Checked  bool   `json:"checked"`
	Selector string `json:"selector"`
}

func cmdCheck(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Check(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return CheckResult{Checked: true, Selector: selector}, nil
	})
}

// UncheckResult is returned by the uncheck command.
type UncheckResult struct {
	Unchecked bool   `json:"unchecked"`
	Selector  string `json:"selector"`
}

func cmdUncheck(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Uncheck(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return UncheckResult{Unchecked: true, Selector: selector}, nil
	})
}

// UploadResult is returned by the upload command.
type UploadResult struct {
	Uploaded bool     `json:"uploaded"`
	Selector string   `json:"selector"`
	Files    []string `json:"files"`
}

func cmdUpload(cfg *Config, selector string, files []string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.UploadFile(ctx, target.ID, selector, files)
		if err != nil {
			return nil, err
		}
		return UploadResult{Uploaded: true, Selector: selector, Files: files}, nil
	})
}

// TripleClickResult is returned by the tripleclick command.
type TripleClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdTripleClick(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.TripleClick(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return TripleClickResult{Clicked: true, Selector: selector}, nil
	})
}

func cmdDispatch(cfg *Config, selector, eventType string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.DispatchEvent(ctx, target.ID, selector, eventType)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

func cmdMouse(cfg *Config, xStr, yStr string) int {
	x, err := strconv.ParseFloat(xStr, 64)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid x coordinate: %v\n", err)
		return ExitError
	}
	y, err := strconv.ParseFloat(yStr, 64)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid y coordinate: %v\n", err)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.MouseMove(ctx, target.ID, x, y)
	})
}

// SwipeCLIResult wraps swipe result for CLI output.
type SwipeCLIResult struct {
	Swiped    bool   `json:"swiped"`
	Direction string `json:"direction"`
	Selector  string `json:"selector"`
}

func cmdSwipe(cfg *Config, selector, direction string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.Swipe(ctx, target.ID, selector, direction)
	})
}

func cmdPinch(cfg *Config, selector, direction string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.Pinch(ctx, target.ID, selector, direction)
	})
}
