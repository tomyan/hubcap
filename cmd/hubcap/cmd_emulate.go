package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
)

// --- EMULATION ---

type EmulateResult struct {
	Device            string  `json:"device"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
	Mobile            bool    `json:"mobile"`
}

func cmdEmulate(cfg *Config, deviceName string) int {
	device, ok := chrome.CommonDevices[deviceName]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: unknown device: %s\n", deviceName)
		fmt.Fprintln(cfg.Stderr, "\nAvailable devices:")
		for name := range chrome.CommonDevices {
			fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.Emulate(ctx, target.ID, device)
		if err != nil {
			return nil, err
		}
		return EmulateResult{
			Device:            device.Name,
			Width:             device.Width,
			Height:            device.Height,
			DeviceScaleFactor: device.DeviceScaleFactor,
			Mobile:            device.Mobile,
		}, nil
	})
}

type UserAgentResult struct {
	UserAgent string `json:"userAgent"`
}

func cmdUserAgent(cfg *Config, userAgent string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetUserAgent(ctx, target.ID, userAgent)
		if err != nil {
			return nil, err
		}
		return UserAgentResult{UserAgent: userAgent}, nil
	})
}

type GeolocationResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy"`
}

func cmdGeolocation(cfg *Config, latStr, lonStr string) int {
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: invalid latitude: %v\n", err)
		return ExitError
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: invalid longitude: %v\n", err)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetGeolocation(ctx, target.ID, lat, lon, 1.0) // accuracy of 1 meter
		if err != nil {
			return nil, err
		}
		return GeolocationResult{Latitude: lat, Longitude: lon, Accuracy: 1.0}, nil
	})
}

type OfflineResult struct {
	Offline bool `json:"offline"`
}

func cmdOffline(cfg *Config, offlineStr string) int {
	offline, err := strconv.ParseBool(offlineStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: invalid value, use 'true' or 'false'\n")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetOfflineMode(ctx, target.ID, offline)
		if err != nil {
			return nil, err
		}
		return OfflineResult{Offline: offline}, nil
	})
}

// MediaResult is returned by the media command.
type MediaResult struct {
	ColorScheme   string `json:"colorScheme,omitempty"`
	ReducedMotion string `json:"reducedMotion,omitempty"`
	ForcedColors  string `json:"forcedColors,omitempty"`
}

func cmdMedia(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("media", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	colorScheme := fs.String("color-scheme", "", "prefers-color-scheme (light, dark)")
	reducedMotion := fs.String("reduced-motion", "", "prefers-reduced-motion (reduce, no-preference)")
	forcedColors := fs.String("forced-colors", "", "forced-colors (active, none)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *colorScheme == "" && *reducedMotion == "" && *forcedColors == "" {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap media [--color-scheme <light|dark>] [--reduced-motion <reduce|no-preference>] [--forced-colors <active|none>]")
		return ExitError
	}

	features := chrome.MediaFeatures{
		ColorScheme:   *colorScheme,
		ReducedMotion: *reducedMotion,
		ForcedColors:  *forcedColors,
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetEmulatedMedia(ctx, target.ID, features)
		if err != nil {
			return nil, err
		}
		return MediaResult{
			ColorScheme:   *colorScheme,
			ReducedMotion: *reducedMotion,
			ForcedColors:  *forcedColors,
		}, nil
	})
}

// PermissionResult is returned by the permission command.
type PermissionResult struct {
	Permission string `json:"permission"`
	State      string `json:"state"`
}

func cmdPermission(cfg *Config, permission, state string) int {
	// Validate state
	validStates := map[string]bool{"granted": true, "denied": true, "prompt": true}
	if !validStates[state] {
		fmt.Fprintf(cfg.Stderr, "error: invalid state %q (use granted, denied, or prompt)\n", state)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetPermission(ctx, target.ID, permission, state)
		if err != nil {
			return nil, err
		}
		return PermissionResult{Permission: permission, State: state}, nil
	})
}

// --- STORAGE ---

func cmdCookies(cfg *Config, args []string) int {
	// Parse cookies-specific flags
	fs := flag.NewFlagSet("cookies", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	setName := fs.String("set", "", "Cookie name=value to set")
	deleteName := fs.String("delete", "", "Cookie name to delete")
	clearAll := fs.Bool("clear", false, "Clear all cookies")
	domain := fs.String("domain", "", "Cookie domain (for set/delete)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *setName != "" {
		// Set mode
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			// Parse name=value
			parts := splitCookieValue(*setName)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid cookie format, use name=value")
			}

			cookie := chrome.Cookie{
				Name:   parts[0],
				Value:  parts[1],
				Domain: *domain,
			}

			err := client.SetCookie(ctx, target.ID, cookie)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"set":    true,
				"name":   cookie.Name,
				"value":  cookie.Value,
				"domain": cookie.Domain,
			}, nil
		})
	}

	if *deleteName != "" {
		// Delete mode
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.DeleteCookie(ctx, target.ID, *deleteName, *domain)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"deleted": true,
				"name":    *deleteName,
				"domain":  *domain,
			}, nil
		})
	}

	if *clearAll {
		// Clear all cookies mode
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.ClearCookies(ctx, target.ID)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"cleared": true,
			}, nil
		})
	}

	// List mode (default)
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetCookies(ctx, target.ID)
	})
}

func splitCookieValue(s string) []string {
	idx := -1
	for i, c := range s {
		if c == '=' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

// StorageResult is returned by storage get command.
type StorageResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// StorageSetResult is returned by storage set command.
type StorageSetResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Set   bool   `json:"set"`
}

// StorageClearResult is returned by storage clear command.
type StorageClearResult struct {
	Cleared bool `json:"cleared"`
}

func cmdStorage(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("storage", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	clear := fs.Bool("clear", false, "Clear all localStorage")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()

	if *clear {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.ClearLocalStorage(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return StorageClearResult{Cleared: true}, nil
		})
	}

	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap storage <key> [value] | --clear")
		return ExitError
	}

	key := remaining[0]

	if len(remaining) == 1 {
		// Get
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			value, err := client.GetLocalStorage(ctx, target.ID, key)
			if err != nil {
				return nil, err
			}
			return StorageResult{Key: key, Value: value}, nil
		})
	}

	// Set
	value := remaining[1]
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetLocalStorage(ctx, target.ID, key, value)
		if err != nil {
			return nil, err
		}
		return StorageSetResult{Key: key, Value: value, Set: true}, nil
	})
}

func cmdSession(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("session", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	clear := fs.Bool("clear", false, "Clear all sessionStorage")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()

	if *clear {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.ClearSessionStorage(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return StorageClearResult{Cleared: true}, nil
		})
	}

	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap session <key> [value] | --clear")
		return ExitError
	}

	key := remaining[0]

	if len(remaining) == 1 {
		// Get
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			value, err := client.GetSessionStorage(ctx, target.ID, key)
			if err != nil {
				return nil, err
			}
			return StorageResult{Key: key, Value: value}, nil
		})
	}

	// Set
	value := remaining[1]
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.SetSessionStorage(ctx, target.ID, key, value)
		if err != nil {
			return nil, err
		}
		return StorageSetResult{Key: key, Value: value, Set: true}, nil
	})
}

// ClipboardWriteResult is returned by clipboard write command.
type ClipboardWriteResult struct {
	Written bool   `json:"written"`
	Text    string `json:"text"`
}

// ClipboardReadResult is returned by clipboard read command.
type ClipboardReadResult struct {
	Text string `json:"text"`
}

func cmdClipboard(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("clipboard", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	write := fs.String("write", "", "Text to write to clipboard")
	read := fs.Bool("read", false, "Read from clipboard")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *write == "" && !*read {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap clipboard --write <text> | --read")
		return ExitError
	}

	if *write != "" {
		return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
			err := client.WriteClipboard(ctx, target.ID, *write)
			if err != nil {
				return nil, err
			}
			return ClipboardWriteResult{Written: true, Text: *write}, nil
		})
	}

	// Read
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		text, err := client.ReadClipboard(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ClipboardReadResult{Text: text}, nil
	})
}

// --- ANALYSIS ---

type MetricsResult struct {
	Metrics map[string]float64 `json:"metrics"`
}

func cmdMetrics(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		metrics, err := client.GetPerformanceMetrics(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return MetricsResult{Metrics: metrics}, nil
	})
}

type A11yResult struct {
	Nodes []chrome.AccessibilityNode `json:"nodes"`
}

func cmdA11y(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		nodes, err := client.GetAccessibilityTree(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return A11yResult{Nodes: nodes}, nil
	})
}

// --- PROFILING ---

// HeapSnapshotCLIResult wraps heap snapshot result for CLI output.
type HeapSnapshotCLIResult struct {
	File string `json:"file"`
	Size int    `json:"size"`
}

func cmdHeapSnapshot(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("heapsnapshot", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	output := fs.String("output", "", "Output file path")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *output == "" {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap heapsnapshot --output <file>")
		return ExitError
	}

	outputFile := *output
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		data, err := client.TakeHeapSnapshot(ctx, target.ID)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return nil, fmt.Errorf("writing file: %w", err)
		}

		return HeapSnapshotCLIResult{
			File: outputFile,
			Size: len(data),
		}, nil
	})
}

// TraceCLIResult wraps trace result for CLI output.
type TraceCLIResult struct {
	File string `json:"file"`
	Size int    `json:"size"`
}

func cmdTrace(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	output := fs.String("output", "", "Output file path")
	duration := fs.Duration("duration", 1*time.Second, "Trace duration")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *output == "" {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap trace --duration <d> --output <file>")
		return ExitError
	}

	outputFile := *output
	traceDuration := *duration
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		data, err := client.CaptureTrace(ctx, target.ID, traceDuration)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return nil, fmt.Errorf("writing file: %w", err)
		}

		return TraceCLIResult{
			File: outputFile,
			Size: len(data),
		}, nil
	})
}

// --- ADVANCED ---

func cmdEval(cfg *Config, expression string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.Eval(ctx, target.ID, expression)
	})
}

// EvalFrameResult is returned by the evalframe command.
type EvalFrameResult struct {
	FrameID string      `json:"frameId"`
	Type    string      `json:"type"`
	Value   interface{} `json:"value"`
}

func cmdEvalFrame(cfg *Config, frameID, expression string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.EvalInFrame(ctx, target.ID, frameID, expression)
		if err != nil {
			return nil, err
		}
		return EvalFrameResult{FrameID: frameID, Type: result.Type, Value: result.Value}, nil
	})
}

// RunResult is returned by the run command.
type RunResult struct {
	File  string      `json:"file"`
	Value interface{} `json:"value,omitempty"`
}

func cmdRun(cfg *Config, file string) int {
	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error reading file: %v\n", err)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.ExecuteScriptFile(ctx, target.ID, string(content))
		if err != nil {
			return nil, err
		}
		return RunResult{File: file, Value: result.Value}, nil
	})
}

func cmdRaw(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("raw", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	browser := fs.Bool("browser", false, "Send command at browser level (not to a page target)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap raw [--browser] <method> [params-json]")
		fmt.Fprintln(cfg.Stderr, "")
		fmt.Fprintln(cfg.Stderr, "examples:")
		fmt.Fprintln(cfg.Stderr, "  hubcap raw Page.navigate '{\"url\":\"https://example.com\"}'")
		fmt.Fprintln(cfg.Stderr, "  hubcap raw Runtime.evaluate '{\"expression\":\"1+1\"}'")
		fmt.Fprintln(cfg.Stderr, "  hubcap raw --browser Target.getTargets")
		fmt.Fprintln(cfg.Stderr, "  hubcap raw DOM.getDocument")
		return ExitError
	}

	method := remaining[0]
	var params json.RawMessage
	if len(remaining) > 1 {
		params = json.RawMessage(remaining[1])
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := chrome.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	var result json.RawMessage

	if *browser {
		// Browser-level command
		result, err = client.RawCall(ctx, method, params)
	} else {
		// Session-level command (to resolved target)
		target, targetErr := resolveTarget(ctx, client, cfg)
		if targetErr != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", targetErr)
			return ExitError
		}
		result, err = client.RawCallSession(ctx, target.ID, method, params)
	}

	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	// Pretty-print the result
	var prettyResult interface{}
	if err := json.Unmarshal(result, &prettyResult); err != nil {
		// If we can't parse it, just output raw
		fmt.Fprintln(cfg.Stdout, string(result))
	} else {
		enc := json.NewEncoder(cfg.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(prettyResult)
	}

	return ExitSuccess
}

// DialogResult is returned by the dialog command.
type DialogResult struct {
	Action     string `json:"action"`
	PromptText string `json:"promptText,omitempty"`
}

func cmdDialog(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("dialog", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	promptText := fs.String("text", "", "Text to enter for prompts")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap dialog [accept|dismiss] [--text <prompt-text>]")
		return ExitError
	}

	action := remaining[0]
	if action != "accept" && action != "dismiss" {
		fmt.Fprintln(cfg.Stderr, "action must be 'accept' or 'dismiss'")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		err := client.HandleDialog(ctx, target.ID, action, *promptText)
		if err != nil {
			return nil, err
		}
		return DialogResult{Action: action, PromptText: *promptText}, nil
	})
}
