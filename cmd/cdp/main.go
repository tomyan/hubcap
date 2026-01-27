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
		fmt.Fprintln(cfg.Stderr, "commands: version, tabs, goto, screenshot, eval, query, click, fill, html, wait, text, type, console, cookies, pdf, focus")
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
