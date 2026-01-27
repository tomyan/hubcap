package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/tomyan/cdp-cli/internal/cdp"
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

	Stdout io.Writer
	Stderr io.Writer
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:    getEnvInt("CDP_PORT", 9222),
		Host:    getEnv("CDP_HOST", "localhost"),
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
	os.Exit(run(os.Args[1:], DefaultConfig()))
}

func run(args []string, cfg *Config) int {
	// Parse global flags
	fs := flag.NewFlagSet("cdp", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "Chrome debug port (env: CDP_PORT)")
	fs.StringVar(&cfg.Host, "host", cfg.Host, "Chrome debug host (env: CDP_HOST)")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Command timeout")
	fs.StringVar(&cfg.Output, "output", cfg.Output, "Output format: json, ndjson, text")
	fs.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "Suppress non-essential output")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: cdp [flags] <command>")
		fmt.Fprintln(cfg.Stderr, "commands: version, tabs, goto, screenshot, eval, query, click, dblclick, rightclick, fill, clear, select, check, uncheck, html, wait, text, type, console, cookies, pdf, focus, network, press, hover, attr, reload, back, forward, title, url, new, close, scrollto, scroll, count, visible, bounds, viewport, waitload, storage, dialog, run")
		fmt.Fprintln(cfg.Stderr, "flags:")
		fs.PrintDefaults()
		return ExitError
	}

	cmd := remaining[0]

	switch cmd {
	case "version":
		return cmdVersion(cfg)
	case "tabs":
		return cmdTabs(cfg)
	case "goto":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp goto <url>")
			return ExitError
		}
		return cmdGoto(cfg, remaining[1])
	case "screenshot":
		return cmdScreenshot(cfg, remaining[1:])
	case "eval":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp eval <expression>")
			return ExitError
		}
		return cmdEval(cfg, remaining[1])
	case "query":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp query <selector>")
			return ExitError
		}
		return cmdQuery(cfg, remaining[1])
	case "click":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp click <selector>")
			return ExitError
		}
		return cmdClick(cfg, remaining[1])
	case "fill":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp fill <selector> <text>")
			return ExitError
		}
		return cmdFill(cfg, remaining[1], remaining[2])
	case "html":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp html <selector>")
			return ExitError
		}
		return cmdHTML(cfg, remaining[1])
	case "wait":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp wait <selector> [--timeout <duration>]")
			return ExitError
		}
		return cmdWait(cfg, remaining[1:])
	case "text":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp text <selector>")
			return ExitError
		}
		return cmdText(cfg, remaining[1])
	case "type":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp type <text>")
			return ExitError
		}
		return cmdType(cfg, remaining[1])
	case "console":
		return cmdConsole(cfg, remaining[1:])
	case "cookies":
		return cmdCookies(cfg, remaining[1:])
	case "pdf":
		return cmdPDF(cfg, remaining[1:])
	case "focus":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp focus <selector>")
			return ExitError
		}
		return cmdFocus(cfg, remaining[1])
	case "network":
		return cmdNetwork(cfg, remaining[1:])
	case "press":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp press <key>")
			fmt.Fprintln(cfg.Stderr, "keys: Enter, Tab, Escape, Backspace, Delete, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Home, End, PageUp, PageDown, Space")
			return ExitError
		}
		return cmdPress(cfg, remaining[1])
	case "hover":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp hover <selector>")
			return ExitError
		}
		return cmdHover(cfg, remaining[1])
	case "attr":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp attr <selector> <attribute>")
			return ExitError
		}
		return cmdAttr(cfg, remaining[1], remaining[2])
	case "reload":
		return cmdReload(cfg, remaining[1:])
	case "back":
		return cmdBack(cfg)
	case "forward":
		return cmdForward(cfg)
	case "title":
		return cmdTitle(cfg)
	case "url":
		return cmdURL(cfg)
	case "new":
		url := ""
		if len(remaining) > 1 {
			url = remaining[1]
		}
		return cmdNew(cfg, url)
	case "close":
		return cmdClose(cfg)
	case "dblclick":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp dblclick <selector>")
			return ExitError
		}
		return cmdDblClick(cfg, remaining[1])
	case "rightclick":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp rightclick <selector>")
			return ExitError
		}
		return cmdRightClick(cfg, remaining[1])
	case "clear":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp clear <selector>")
			return ExitError
		}
		return cmdClear(cfg, remaining[1])
	case "select":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp select <selector> <value>")
			return ExitError
		}
		return cmdSelect(cfg, remaining[1], remaining[2])
	case "check":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp check <selector>")
			return ExitError
		}
		return cmdCheck(cfg, remaining[1])
	case "uncheck":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp uncheck <selector>")
			return ExitError
		}
		return cmdUncheck(cfg, remaining[1])
	case "scrollto":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp scrollto <selector>")
			return ExitError
		}
		return cmdScrollTo(cfg, remaining[1])
	case "scroll":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp scroll <x> <y>")
			return ExitError
		}
		return cmdScroll(cfg, remaining[1], remaining[2])
	case "count":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp count <selector>")
			return ExitError
		}
		return cmdCount(cfg, remaining[1])
	case "visible":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp visible <selector>")
			return ExitError
		}
		return cmdVisible(cfg, remaining[1])
	case "bounds":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp bounds <selector>")
			return ExitError
		}
		return cmdBounds(cfg, remaining[1])
	case "viewport":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp viewport <width> <height>")
			return ExitError
		}
		return cmdViewport(cfg, remaining[1], remaining[2])
	case "waitload":
		return cmdWaitLoad(cfg, remaining[1:])
	case "storage":
		return cmdStorage(cfg, remaining[1:])
	case "dialog":
		return cmdDialog(cfg, remaining[1:])
	case "run":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp run <file.js>")
			return ExitError
		}
		return cmdRun(cfg, remaining[1])
	default:
		fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", cmd)
		return ExitError
	}
}

// withClient executes a function with a connected CDP client.
func withClient(cfg *Config, fn func(ctx context.Context, client *cdp.Client) (interface{}, error)) int {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
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

func cmdVersion(cfg *Config) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		return client.Version(ctx)
	})
}

func cmdTabs(cfg *Config) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		return client.Pages(ctx)
	})
}

func cmdGoto(cfg *Config, url string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		// Get first page to navigate
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		return client.Navigate(ctx, pages[0].ID, url)
	})
}

func cmdScreenshot(cfg *Config, args []string) int {
	// Parse screenshot-specific flags
	fs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	output := fs.String("output", "", "Output file path (required)")
	format := fs.String("format", "png", "Image format: png, jpeg, webp")
	quality := fs.Int("quality", 80, "JPEG/WebP quality (0-100)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *output == "" {
		fmt.Fprintln(cfg.Stderr, "usage: cdp screenshot --output <file> [--format png|jpeg|webp] [--quality 0-100]")
		return ExitError
	}

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		data, err := client.Screenshot(ctx, pages[0].ID, cdp.ScreenshotOptions{
			Format:  *format,
			Quality: *quality,
		})
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(*output, data, 0644); err != nil {
			return nil, fmt.Errorf("writing file: %w", err)
		}

		return cdp.ScreenshotResult{
			Format: *format,
			Size:   len(data),
		}, nil
	})
}

func cmdEval(cfg *Config, expression string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		return client.Eval(ctx, pages[0].ID, expression)
	})
}

func cmdQuery(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		return client.Query(ctx, pages[0].ID, selector)
	})
}

// ClickResult is returned by the click command.
type ClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdClick(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.Click(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}

		return ClickResult{Clicked: true, Selector: selector}, nil
	})
}

// FillResult is returned by the fill command.
type FillResult struct {
	Filled   bool   `json:"filled"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

func cmdFill(cfg *Config, selector, text string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.Fill(ctx, pages[0].ID, selector, text)
		if err != nil {
			return nil, err
		}

		return FillResult{Filled: true, Selector: selector, Text: text}, nil
	})
}

// HTMLResult is returned by the html command.
type HTMLResult struct {
	Selector string `json:"selector"`
	HTML     string `json:"html"`
}

func cmdHTML(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		html, err := client.GetHTML(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}

		return HTMLResult{Selector: selector, HTML: html}, nil
	})
}

// WaitResult is returned by the wait command.
type WaitResult struct {
	Found    bool   `json:"found"`
	Selector string `json:"selector"`
}

// TextResult is returned by the text command.
type TextResult struct {
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

func cmdText(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		text, err := client.GetText(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}

		return TextResult{Selector: selector, Text: text}, nil
	})
}

// TypeResult is returned by the type command.
type TypeResult struct {
	Typed bool   `json:"typed"`
	Text  string `json:"text"`
}

func cmdType(cfg *Config, text string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.Type(ctx, pages[0].ID, text)
		if err != nil {
			return nil, err
		}

		return TypeResult{Typed: true, Text: text}, nil
	})
}

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

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	if len(pages) == 0 {
		fmt.Fprintln(cfg.Stderr, "error: no pages available")
		return ExitError
	}

	messages, err := client.CaptureConsole(ctx, pages[0].ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

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

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	if len(pages) == 0 {
		fmt.Fprintln(cfg.Stderr, "error: no pages available")
		return ExitError
	}

	events, err := client.CaptureNetwork(ctx, pages[0].ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

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
		return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
			pages, err := client.Pages(ctx)
			if err != nil {
				return nil, err
			}
			if len(pages) == 0 {
				return nil, fmt.Errorf("no pages available")
			}

			// Parse name=value
			parts := splitCookieValue(*setName)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid cookie format, use name=value")
			}

			cookie := cdp.Cookie{
				Name:   parts[0],
				Value:  parts[1],
				Domain: *domain,
			}

			err = client.SetCookie(ctx, pages[0].ID, cookie)
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
		return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
			pages, err := client.Pages(ctx)
			if err != nil {
				return nil, err
			}
			if len(pages) == 0 {
				return nil, fmt.Errorf("no pages available")
			}

			err = client.DeleteCookie(ctx, pages[0].ID, *deleteName, *domain)
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
		return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
			pages, err := client.Pages(ctx)
			if err != nil {
				return nil, err
			}
			if len(pages) == 0 {
				return nil, fmt.Errorf("no pages available")
			}

			err = client.ClearCookies(ctx, pages[0].ID)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"cleared": true,
			}, nil
		})
	}

	// List mode (default)
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		return client.GetCookies(ctx, pages[0].ID)
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

func cmdPDF(cfg *Config, args []string) int {
	// Parse pdf-specific flags
	fs := flag.NewFlagSet("pdf", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	output := fs.String("output", "", "Output file path (required)")
	landscape := fs.Bool("landscape", false, "Landscape orientation")
	background := fs.Bool("background", false, "Print background graphics")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *output == "" {
		fmt.Fprintln(cfg.Stderr, "usage: cdp pdf --output <file> [--landscape] [--background]")
		return ExitError
	}

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		data, err := client.PrintToPDF(ctx, pages[0].ID, cdp.PDFOptions{
			Landscape:       *landscape,
			PrintBackground: *background,
		})
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(*output, data, 0644); err != nil {
			return nil, fmt.Errorf("writing file: %w", err)
		}

		return map[string]interface{}{
			"output":    *output,
			"size":      len(data),
			"landscape": *landscape,
		}, nil
	})
}

// FocusResult is returned by the focus command.
type FocusResult struct {
	Focused  bool   `json:"focused"`
	Selector string `json:"selector"`
}

func cmdFocus(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.Focus(ctx, pages[0].ID, selector)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.PressKey(ctx, pages[0].ID, key)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.Hover(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}

		return HoverResult{Hovered: true, Selector: selector}, nil
	})
}

// AttrResult is returned by the attr command.
type AttrResult struct {
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

func cmdAttr(cfg *Config, selector, attribute string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		value, err := client.GetAttribute(ctx, pages[0].ID, selector, attribute)
		if err != nil {
			return nil, err
		}

		return AttrResult{Selector: selector, Attribute: attribute, Value: value}, nil
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

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.Reload(ctx, pages[0].ID, *ignoreCache)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.GoBack(ctx, pages[0].ID)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.GoForward(ctx, pages[0].ID)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		title, err := client.GetTitle(ctx, pages[0].ID)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		url, err := client.GetURL(ctx, pages[0].ID)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		targetID := pages[0].ID
		err = client.CloseTab(ctx, targetID)
		if err != nil {
			return nil, err
		}

		return CloseTabResult{Closed: true, TargetID: targetID}, nil
	})
}

// DblClickResult is returned by the dblclick command.
type DblClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdDblClick(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.DoubleClick(ctx, pages[0].ID, selector)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.RightClick(ctx, pages[0].ID, selector)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.Clear(ctx, pages[0].ID, selector)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.SelectOption(ctx, pages[0].ID, selector, value)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.Check(ctx, pages[0].ID, selector)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.Uncheck(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}
		return UncheckResult{Unchecked: true, Selector: selector}, nil
	})
}

// ScrollToResult is returned by the scrollto command.
type ScrollToResult struct {
	Scrolled bool   `json:"scrolled"`
	Selector string `json:"selector"`
}

func cmdScrollTo(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.ScrollIntoView(ctx, pages[0].ID, selector)
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

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.ScrollBy(ctx, pages[0].ID, x, y)
		if err != nil {
			return nil, err
		}
		return ScrollResult{Scrolled: true, X: x, Y: y}, nil
	})
}

// CountResult is returned by the count command.
type CountResult struct {
	Count    int    `json:"count"`
	Selector string `json:"selector"`
}

func cmdCount(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		count, err := client.CountElements(ctx, pages[0].ID, selector)
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
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		visible, err := client.IsVisible(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}
		return VisibleResult{Visible: visible, Selector: selector}, nil
	})
}

func cmdBounds(cfg *Config, selector string) int {
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		bounds, err := client.GetBoundingBox(ctx, pages[0].ID, selector)
		if err != nil {
			return nil, err
		}
		return bounds, nil
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

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.SetViewport(ctx, pages[0].ID, width, height)
		if err != nil {
			return nil, err
		}
		return ViewportResult{Width: width, Height: height}, nil
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

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	if len(pages) == 0 {
		fmt.Fprintln(cfg.Stderr, "error: no pages available")
		return ExitError
	}

	err = client.WaitForLoad(ctx, pages[0].ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	result := WaitLoadResult{Loaded: true}
	enc := json.NewEncoder(cfg.Stdout)
	enc.Encode(result)
	return ExitSuccess
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
		return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
			pages, err := client.Pages(ctx)
			if err != nil {
				return nil, err
			}
			if len(pages) == 0 {
				return nil, fmt.Errorf("no pages available")
			}
			err = client.ClearLocalStorage(ctx, pages[0].ID)
			if err != nil {
				return nil, err
			}
			return StorageClearResult{Cleared: true}, nil
		})
	}

	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: cdp storage <key> [value] | --clear")
		return ExitError
	}

	key := remaining[0]

	if len(remaining) == 1 {
		// Get
		return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
			pages, err := client.Pages(ctx)
			if err != nil {
				return nil, err
			}
			if len(pages) == 0 {
				return nil, fmt.Errorf("no pages available")
			}
			value, err := client.GetLocalStorage(ctx, pages[0].ID, key)
			if err != nil {
				return nil, err
			}
			return StorageResult{Key: key, Value: value}, nil
		})
	}

	// Set
	value := remaining[1]
	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}
		err = client.SetLocalStorage(ctx, pages[0].ID, key, value)
		if err != nil {
			return nil, err
		}
		return StorageSetResult{Key: key, Value: value, Set: true}, nil
	})
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp dialog [accept|dismiss] [--text <prompt-text>]")
		return ExitError
	}

	action := remaining[0]
	if action != "accept" && action != "dismiss" {
		fmt.Fprintln(cfg.Stderr, "action must be 'accept' or 'dismiss'")
		return ExitError
	}

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.HandleDialog(ctx, pages[0].ID, action, *promptText)
		if err != nil {
			return nil, err
		}

		return DialogResult{Action: action, PromptText: *promptText}, nil
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

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		result, err := client.ExecuteScriptFile(ctx, pages[0].ID, string(content))
		if err != nil {
			return nil, err
		}

		return RunResult{File: file, Value: result.Value}, nil
	})
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp wait <selector> [--timeout <duration>]")
		return ExitError
	}
	selector := remaining[0]

	return withClient(cfg, func(ctx context.Context, client *cdp.Client) (interface{}, error) {
		pages, err := client.Pages(ctx)
		if err != nil {
			return nil, err
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("no pages available")
		}

		err = client.WaitFor(ctx, pages[0].ID, selector, *timeout)
		if err != nil {
			return nil, err
		}

		return WaitResult{Found: true, Selector: selector}, nil
	})
}

func outputResult(cfg *Config, v interface{}) int {
	switch cfg.Output {
	case "json":
		enc := json.NewEncoder(cfg.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
	case "ndjson":
		enc := json.NewEncoder(cfg.Stdout)
		if err := enc.Encode(v); err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
	case "text":
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
		fmt.Fprintln(cfg.Stdout, string(data))
	default:
		fmt.Fprintf(cfg.Stderr, "error: unknown output format: %s\n", cfg.Output)
		return ExitError
	}
	return ExitSuccess
}
