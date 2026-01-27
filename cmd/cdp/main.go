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
		fmt.Fprintln(cfg.Stderr, "commands: version, tabs, goto <url>, screenshot --output <file>, eval <expr>")
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
