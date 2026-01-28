package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	Target  string // target index or ID

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
	fs.StringVar(&cfg.Target, "target", cfg.Target, "Target page (index or ID)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: cdp [flags] <command>")
		fmt.Fprintln(cfg.Stderr, "commands: version, tabs, goto, screenshot, eval, query, click, dblclick, rightclick, fill, clear, select, check, uncheck, html, wait, text, type, console, cookies, pdf, focus, network, press, hover, attr, reload, back, forward, title, url, new, close, scrollto, scroll, count, visible, bounds, viewport, waitload, storage, dialog, run, raw, upload, exists, waitnav, value, waitfn, forms, highlight, images, scrollbottom, scrolltop, frames, evalframe, waitgone")
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
	case "clickat":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp clickat <x> <y>")
			return ExitError
		}
		return cmdClickAt(cfg, remaining[1], remaining[2])
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
	case "errors":
		return cmdErrors(cfg, remaining[1:])
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
	case "tap":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp tap <selector>")
			return ExitError
		}
		return cmdTap(cfg, remaining[1])
	case "drag":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp drag <source-selector> <dest-selector>")
			return ExitError
		}
		return cmdDrag(cfg, remaining[1], remaining[2])
	case "waiturl":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp waiturl <pattern> [--timeout <duration>]")
			return ExitError
		}
		return cmdWaitURL(cfg, remaining[1], remaining[2:])
	case "shadow":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp shadow <host-selector> <inner-selector>")
			return ExitError
		}
		return cmdShadow(cfg, remaining[1], remaining[2])
	case "har":
		return cmdHar(cfg, remaining[1:])
	case "coverage":
		return cmdCoverage(cfg)
	case "stylesheets":
		return cmdStylesheets(cfg)
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
	case "session":
		return cmdSession(cfg, remaining[1:])
	case "dialog":
		return cmdDialog(cfg, remaining[1:])
	case "run":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp run <file.js>")
			return ExitError
		}
		return cmdRun(cfg, remaining[1])
	case "raw":
		return cmdRaw(cfg, remaining[1:])
	case "emulate":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp emulate <device>")
			fmt.Fprintln(cfg.Stderr, "\nAvailable devices:")
			for name := range cdp.CommonDevices {
				fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
			}
			return ExitError
		}
		return cmdEmulate(cfg, remaining[1])
	case "useragent":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp useragent <string>")
			return ExitError
		}
		return cmdUserAgent(cfg, remaining[1])
	case "geolocation":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp geolocation <latitude> <longitude>")
			return ExitError
		}
		return cmdGeolocation(cfg, remaining[1], remaining[2])
	case "offline":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp offline <true|false>")
			return ExitError
		}
		return cmdOffline(cfg, remaining[1])
	case "throttle":
		return cmdThrottle(cfg, remaining[1:])
	case "media":
		return cmdMedia(cfg, remaining[1:])
	case "permission":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp permission <name> <granted|denied|prompt>")
			fmt.Fprintln(cfg.Stderr, "\nPermission names: geolocation, notifications, camera, microphone, midi, push")
			return ExitError
		}
		return cmdPermission(cfg, remaining[1], remaining[2])
	case "clipboard":
		return cmdClipboard(cfg, remaining[1:])
	case "styles":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp styles <selector>")
			return ExitError
		}
		return cmdStyles(cfg, remaining[1])
	case "layout":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp layout <selector> [--depth <n>]")
			return ExitError
		}
		return cmdLayout(cfg, remaining[1:])
	case "intercept":
		return cmdIntercept(cfg, remaining[1:])
	case "block":
		return cmdBlock(cfg, remaining[1:])
	case "metrics":
		return cmdMetrics(cfg)
	case "a11y":
		return cmdA11y(cfg)
	case "source":
		return cmdSource(cfg)
	case "waitidle":
		return cmdWaitIdle(cfg, remaining[1:])
	case "links":
		return cmdLinks(cfg)
	case "meta":
		return cmdMeta(cfg)
	case "tables":
		return cmdTables(cfg)
	case "upload":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp upload <selector> <file>...")
			return ExitError
		}
		return cmdUpload(cfg, remaining[1], remaining[2:])
	case "exists":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp exists <selector>")
			return ExitError
		}
		return cmdExists(cfg, remaining[1])
	case "waitnav":
		return cmdWaitNav(cfg, remaining[1:])
	case "value":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp value <selector>")
			return ExitError
		}
		return cmdValue(cfg, remaining[1])
	case "waitfn":
		return cmdWaitFn(cfg, remaining[1:])
	case "forms":
		return cmdForms(cfg)
	case "highlight":
		if len(remaining) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp highlight <selector> [--hide]")
			return ExitError
		}
		return cmdHighlight(cfg, remaining[1:])
	case "images":
		return cmdImages(cfg)
	case "scrollbottom":
		return cmdScrollBottom(cfg)
	case "scrolltop":
		return cmdScrollTop(cfg)
	case "frames":
		return cmdFrames(cfg)
	case "evalframe":
		if len(remaining) < 3 {
			fmt.Fprintln(cfg.Stderr, "usage: cdp evalframe <frame-id> <expression>")
			return ExitError
		}
		return cmdEvalFrame(cfg, remaining[1], remaining[2])
	case "waitgone":
		return cmdWaitGone(cfg, remaining[1:])
	default:
		fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", cmd)
		return ExitError
	}
}

// resolveTarget resolves the target page from cfg.Target.
// If cfg.Target is empty, returns the first page.
// If cfg.Target is a number, uses it as an index into the pages list.
// Otherwise, treats cfg.Target as a target ID.
func resolveTarget(ctx context.Context, client *cdp.Client, cfg *Config) (*cdp.TargetInfo, error) {
	pages, err := client.Pages(ctx)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages available")
	}

	// Default: first page
	if cfg.Target == "" {
		return &pages[0], nil
	}

	// Try as index first
	if idx, err := strconv.Atoi(cfg.Target); err == nil {
		if idx < 0 || idx >= len(pages) {
			return nil, fmt.Errorf("invalid target index: %d (have %d pages)", idx, len(pages))
		}
		return &pages[idx], nil
	}

	// Otherwise, treat as target ID
	for i := range pages {
		if pages[i].ID == cfg.Target {
			return &pages[i], nil
		}
	}

	return nil, fmt.Errorf("invalid target: %s (not found)", cfg.Target)
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

// withClientTarget executes a function with a connected CDP client and resolved target.
func withClientTarget(cfg *Config, fn func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error)) int {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
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

	result, err := fn(ctx, client, target)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		return client.Navigate(ctx, target.ID, url)
	})
}

// ElementScreenshotResult contains metadata about an element screenshot.
type ElementScreenshotResult struct {
	Format   string         `json:"format"`
	Size     int            `json:"size"`
	Selector string         `json:"selector,omitempty"`
	Bounds   *cdp.BoundingBox `json:"bounds,omitempty"`
}

// Base64ScreenshotResult is returned by the screenshot command with --base64.
type Base64ScreenshotResult struct {
	Format string `json:"format"`
	Size   int    `json:"size"`
	Data   string `json:"data"`
}

func cmdScreenshot(cfg *Config, args []string) int {
	// Parse screenshot-specific flags
	fs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	output := fs.String("output", "", "Output file path")
	format := fs.String("format", "png", "Image format: png, jpeg, webp")
	quality := fs.Int("quality", 80, "JPEG/WebP quality (0-100)")
	selector := fs.String("selector", "", "CSS selector for element screenshot")
	base64Flag := fs.Bool("base64", false, "Return base64 data instead of writing to file")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	if *output == "" && !*base64Flag {
		fmt.Fprintln(cfg.Stderr, "usage: cdp screenshot --output <file> [--format png|jpeg|webp] [--quality 0-100] [--selector <css>]")
		fmt.Fprintln(cfg.Stderr, "       cdp screenshot --base64 [--format png|jpeg|webp] [--quality 0-100]")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		opts := cdp.ScreenshotOptions{
			Format:  *format,
			Quality: *quality,
		}

		var data []byte
		var bounds *cdp.BoundingBox
		var err error

		if *selector != "" {
			// Element-specific screenshot
			data, bounds, err = client.ScreenshotElement(ctx, target.ID, *selector, opts)
		} else {
			// Full page screenshot
			data, err = client.Screenshot(ctx, target.ID, opts)
		}

		if err != nil {
			return nil, err
		}

		// Return base64 if requested
		if *base64Flag {
			return Base64ScreenshotResult{
				Format: *format,
				Size:   len(data),
				Data:   base64.StdEncoding.EncodeToString(data),
			}, nil
		}

		if err := os.WriteFile(*output, data, 0644); err != nil {
			return nil, fmt.Errorf("writing file: %w", err)
		}

		if *selector != "" {
			return ElementScreenshotResult{
				Format:   *format,
				Size:     len(data),
				Selector: *selector,
				Bounds:   bounds,
			}, nil
		}

		return cdp.ScreenshotResult{
			Format: *format,
			Size:   len(data),
		}, nil
	})
}

func cmdEval(cfg *Config, expression string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		return client.Eval(ctx, target.ID, expression)
	})
}

func cmdQuery(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		return client.Query(ctx, target.ID, selector)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

// ClickResult is returned by the click command.
type ClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdClick(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.Fill(ctx, target.ID, selector, text)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		html, err := client.GetHTML(ctx, target.ID, selector)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		text, err := client.GetText(ctx, target.ID, selector)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.Type(ctx, target.ID, text)
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

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
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

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		return client.CaptureHAR(ctx, target.ID, *duration)
	})
}

func cmdCoverage(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		return client.GetCoverage(ctx, target.ID)
	})
}

func cmdStylesheets(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		return client.GetStylesheets(ctx, target.ID)
	})
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		data, err := client.PrintToPDF(ctx, target.ID, cdp.PDFOptions{
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.PressKey(ctx, target.ID, key)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.Drag(ctx, target.ID, source, dest)
		if err != nil {
			return nil, err
		}
		return DragResult{Dragged: true, Source: source, Dest: dest}, nil
	})
}

// AttrResult is returned by the attr command.
type AttrResult struct {
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

func cmdAttr(cfg *Config, selector, attribute string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		value, err := client.GetAttribute(ctx, target.ID, selector, attribute)
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.CloseTab(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return CloseTabResult{Closed: true, TargetID: target.ID}, nil
	})
}

// DblClickResult is returned by the dblclick command.
type DblClickResult struct {
	Clicked  bool   `json:"clicked"`
	Selector string `json:"selector"`
}

func cmdDblClick(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.Uncheck(ctx, target.ID, selector)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.ScrollBy(ctx, target.ID, x, y)
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		visible, err := client.IsVisible(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return VisibleResult{Visible: visible, Selector: selector}, nil
	})
}

func cmdBounds(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		bounds, err := client.GetBoundingBox(ctx, target.ID, selector)
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.SetViewport(ctx, target.ID, width, height)
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			err := client.ClearLocalStorage(ctx, target.ID)
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			value, err := client.GetLocalStorage(ctx, target.ID, key)
			if err != nil {
				return nil, err
			}
			return StorageResult{Key: key, Value: value}, nil
		})
	}

	// Set
	value := remaining[1]
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			err := client.ClearSessionStorage(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return StorageClearResult{Cleared: true}, nil
		})
	}

	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: cdp session <key> [value] | --clear")
		return ExitError
	}

	key := remaining[0]

	if len(remaining) == 1 {
		// Get
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			value, err := client.GetSessionStorage(ctx, target.ID, key)
			if err != nil {
				return nil, err
			}
			return StorageResult{Key: key, Value: value}, nil
		})
	}

	// Set
	value := remaining[1]
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.SetSessionStorage(ctx, target.ID, key, value)
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.HandleDialog(ctx, target.ID, action, *promptText)
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		result, err := client.ExecuteScriptFile(ctx, target.ID, string(content))
		if err != nil {
			return nil, err
		}
		return RunResult{File: file, Value: result.Value}, nil
	})
}

type EmulateResult struct {
	Device            string  `json:"device"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
	Mobile            bool    `json:"mobile"`
}

type UserAgentResult struct {
	UserAgent string `json:"userAgent"`
}

type GeolocationResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy"`
}

type OfflineResult struct {
	Offline bool `json:"offline"`
}

type InterceptResult struct {
	Enabled     bool   `json:"enabled"`
	Pattern     string `json:"pattern,omitempty"`
	Response    bool   `json:"response,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

type BlockResult struct {
	Enabled  bool     `json:"enabled"`
	Patterns []string `json:"patterns,omitempty"`
}

type MetricsResult struct {
	Metrics map[string]float64 `json:"metrics"`
}

type A11yResult struct {
	Nodes []cdp.AccessibilityNode `json:"nodes"`
}

type SourceResult struct {
	HTML string `json:"html"`
}

type WaitIdleResult struct {
	Idle bool `json:"idle"`
}

type LinkInfo struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

type LinksResult struct {
	Links []LinkInfo `json:"links"`
}

// MetaInfo represents a single meta tag.
type MetaInfo struct {
	Name       string `json:"name,omitempty"`
	Property   string `json:"property,omitempty"`
	Content    string `json:"content,omitempty"`
	Charset    string `json:"charset,omitempty"`
	HTTPEquiv  string `json:"httpEquiv,omitempty"`
}

// MetaResult is returned by the meta command.
type MetaResult struct {
	Tags []MetaInfo `json:"tags"`
}

// TableInfo represents a single table.
type TableInfo struct {
	ID      string     `json:"id,omitempty"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// TablesResult is returned by the tables command.
type TablesResult struct {
	Tables []TableInfo `json:"tables"`
}

type StylesResult struct {
	Selector string            `json:"selector"`
	Styles   map[string]string `json:"styles"`
}

func cmdStyles(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		styles, err := client.GetComputedStyles(ctx, target.ID, selector, nil)
		if err != nil {
			return nil, err
		}
		return StylesResult{Selector: selector, Styles: styles}, nil
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp layout <selector> [--depth <n>]")
		return ExitError
	}

	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		layout, err := client.GetElementLayout(ctx, target.ID, selector, *depth)
		if err != nil {
			return nil, err
		}
		return layout, nil
	})
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		config := cdp.InterceptConfig{
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

func cmdMetrics(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		metrics, err := client.GetPerformanceMetrics(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return MetricsResult{Metrics: metrics}, nil
	})
}

func cmdA11y(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		nodes, err := client.GetAccessibilityTree(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return A11yResult{Nodes: nodes}, nil
	})
}

func cmdSource(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		html, err := client.GetPageSource(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return SourceResult{HTML: html}, nil
	})
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.WaitForNetworkIdle(ctx, target.ID, *idleTime)
		if err != nil {
			return nil, err
		}
		return WaitIdleResult{Idle: true}, nil
	})
}

func cmdLinks(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		result, err := client.Eval(ctx, target.ID, `
			Array.from(document.querySelectorAll('a[href]')).map(a => ({
				href: a.href,
				text: a.textContent.trim()
			}))
		`)
		if err != nil {
			return nil, err
		}

		// Parse the result
		links := []LinkInfo{}
		if arr, ok := result.Value.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					link := LinkInfo{
						Href: fmt.Sprintf("%v", m["href"]),
						Text: fmt.Sprintf("%v", m["text"]),
					}
					links = append(links, link)
				}
			}
		}

		return LinksResult{Links: links}, nil
	})
}

func cmdMeta(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		result, err := client.Eval(ctx, target.ID, `
			Array.from(document.querySelectorAll('meta')).map(m => ({
				name: m.getAttribute('name') || '',
				property: m.getAttribute('property') || '',
				content: m.getAttribute('content') || '',
				charset: m.getAttribute('charset') || '',
				httpEquiv: m.getAttribute('http-equiv') || ''
			}))
		`)
		if err != nil {
			return nil, err
		}

		// Parse the result
		tags := []MetaInfo{}
		if arr, ok := result.Value.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					tag := MetaInfo{
						Name:      fmt.Sprintf("%v", m["name"]),
						Property:  fmt.Sprintf("%v", m["property"]),
						Content:   fmt.Sprintf("%v", m["content"]),
						Charset:   fmt.Sprintf("%v", m["charset"]),
						HTTPEquiv: fmt.Sprintf("%v", m["httpEquiv"]),
					}
					tags = append(tags, tag)
				}
			}
		}

		return MetaResult{Tags: tags}, nil
	})
}

func cmdTables(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		result, err := client.Eval(ctx, target.ID, `
			Array.from(document.querySelectorAll('table')).map(table => {
				const headers = Array.from(table.querySelectorAll('thead th, thead td, tr:first-child th')).map(th => th.textContent.trim());
				const bodyRows = table.querySelectorAll('tbody tr');
				const rows = Array.from(bodyRows.length > 0 ? bodyRows : table.querySelectorAll('tr')).map(tr => {
					// Skip header row if no tbody
					if (bodyRows.length === 0 && tr.querySelector('th')) return null;
					return Array.from(tr.querySelectorAll('td, th')).map(cell => cell.textContent.trim());
				}).filter(r => r !== null);
				return {
					id: table.id || '',
					headers: headers,
					rows: rows
				};
			})
		`)
		if err != nil {
			return nil, err
		}

		// Parse the result
		tables := []TableInfo{}
		if arr, ok := result.Value.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					tableInfo := TableInfo{
						ID:      fmt.Sprintf("%v", m["id"]),
						Headers: []string{},
						Rows:    [][]string{},
					}

					// Parse headers
					if headers, ok := m["headers"].([]interface{}); ok {
						for _, h := range headers {
							tableInfo.Headers = append(tableInfo.Headers, fmt.Sprintf("%v", h))
						}
					}

					// Parse rows
					if rows, ok := m["rows"].([]interface{}); ok {
						for _, row := range rows {
							if cells, ok := row.([]interface{}); ok {
								rowData := []string{}
								for _, cell := range cells {
									rowData = append(rowData, fmt.Sprintf("%v", cell))
								}
								tableInfo.Rows = append(tableInfo.Rows, rowData)
							}
						}
					}

					tables = append(tables, tableInfo)
				}
			}
		}

		return TablesResult{Tables: tables}, nil
	})
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp block <pattern>... [--disable]")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

func cmdOffline(cfg *Config, offlineStr string) int {
	offline, err := strconv.ParseBool(offlineStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: invalid value, use 'true' or 'false'\n")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.SetOfflineMode(ctx, target.ID, offline)
		if err != nil {
			return nil, err
		}
		return OfflineResult{Offline: offline}, nil
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			err := client.DisableNetworkThrottling(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return ThrottleDisabledResult{Disabled: true}, nil
		})
	}

	if len(remaining) == 0 {
		fmt.Fprintln(cfg.Stderr, "usage: cdp throttle <preset> | --disable")
		fmt.Fprintln(cfg.Stderr, "\nAvailable presets:")
		for name := range cdp.NetworkPresets {
			fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
		}
		return ExitError
	}

	preset := remaining[0]
	conditions, ok := cdp.NetworkPresets[preset]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: unknown preset %q\n", preset)
		fmt.Fprintln(cfg.Stderr, "\nAvailable presets:")
		for name := range cdp.NetworkPresets {
			fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.EmulateNetworkConditions(ctx, target.ID, conditions)
		if err != nil {
			return nil, err
		}
		return ThrottleResult{Preset: preset, Enabled: true}, nil
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp media [--color-scheme <light|dark>] [--reduced-motion <reduce|no-preference>] [--forced-colors <active|none>]")
		return ExitError
	}

	features := cdp.MediaFeatures{
		ColorScheme:   *colorScheme,
		ReducedMotion: *reducedMotion,
		ForcedColors:  *forcedColors,
	}

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.SetPermission(ctx, target.ID, permission, state)
		if err != nil {
			return nil, err
		}
		return PermissionResult{Permission: permission, State: state}, nil
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp clipboard --write <text> | --read")
		return ExitError
	}

	if *write != "" {
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			err := client.WriteClipboard(ctx, target.ID, *write)
			if err != nil {
				return nil, err
			}
			return ClipboardWriteResult{Written: true, Text: *write}, nil
		})
	}

	// Read
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		text, err := client.ReadClipboard(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ClipboardReadResult{Text: text}, nil
	})
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.SetGeolocation(ctx, target.ID, lat, lon, 1.0) // accuracy of 1 meter
		if err != nil {
			return nil, err
		}
		return GeolocationResult{Latitude: lat, Longitude: lon, Accuracy: 1.0}, nil
	})
}

func cmdUserAgent(cfg *Config, userAgent string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.SetUserAgent(ctx, target.ID, userAgent)
		if err != nil {
			return nil, err
		}
		return UserAgentResult{UserAgent: userAgent}, nil
	})
}

func cmdEmulate(cfg *Config, deviceName string) int {
	device, ok := cdp.CommonDevices[deviceName]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: unknown device: %s\n", deviceName)
		fmt.Fprintln(cfg.Stderr, "\nAvailable devices:")
		for name := range cdp.CommonDevices {
			fmt.Fprintf(cfg.Stderr, "  - %s\n", name)
		}
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp raw [--browser] <method> [params-json]")
		fmt.Fprintln(cfg.Stderr, "")
		fmt.Fprintln(cfg.Stderr, "examples:")
		fmt.Fprintln(cfg.Stderr, "  cdp raw Page.navigate '{\"url\":\"https://example.com\"}'")
		fmt.Fprintln(cfg.Stderr, "  cdp raw Runtime.evaluate '{\"expression\":\"1+1\"}'")
		fmt.Fprintln(cfg.Stderr, "  cdp raw --browser Target.getTargets")
		fmt.Fprintln(cfg.Stderr, "  cdp raw DOM.getDocument")
		return ExitError
	}

	method := remaining[0]
	var params json.RawMessage
	if len(remaining) > 1 {
		params = json.RawMessage(remaining[1])
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := cdp.Connect(ctx, cfg.Host, cfg.Port)
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.WaitFor(ctx, target.ID, selector, *timeout)
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

// UploadResult is returned by the upload command.
type UploadResult struct {
	Uploaded bool     `json:"uploaded"`
	Selector string   `json:"selector"`
	Files    []string `json:"files"`
}

func cmdUpload(cfg *Config, selector string, files []string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.UploadFile(ctx, target.ID, selector, files)
		if err != nil {
			return nil, err
		}
		return UploadResult{Uploaded: true, Selector: selector, Files: files}, nil
	})
}

// ExistsResult is returned by the exists command.
type ExistsResult struct {
	Exists   bool   `json:"exists"`
	Selector string `json:"selector"`
}

func cmdExists(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		exists, err := client.Exists(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ExistsResult{Exists: exists, Selector: selector}, nil
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.WaitForNavigation(ctx, target.ID, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitNavResult{Navigated: true}, nil
	})
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

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		url, err := client.WaitForURL(ctx, target.ID, pattern, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitURLResult{Pattern: pattern, URL: url}, nil
	})
}

// ValueResult is returned by the value command.
type ValueResult struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

func cmdValue(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		value, err := client.GetValue(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return ValueResult{Selector: selector, Value: value}, nil
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp waitfn <expression> [--timeout <duration>]")
		return ExitError
	}
	expression := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.WaitForFunction(ctx, target.ID, expression, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitFnResult{Completed: true, Expression: expression}, nil
	})
}

// FormsResult is returned by the forms command.
type FormsResult struct {
	Forms []cdp.FormInfo `json:"forms"`
	Count int            `json:"count"`
}

func cmdForms(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		forms, err := client.GetForms(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return FormsResult{Forms: forms, Count: len(forms)}, nil
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
		return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
			err := client.HideHighlight(ctx, target.ID)
			if err != nil {
				return nil, err
			}
			return HighlightResult{Hidden: true}, nil
		})
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: cdp highlight <selector> [--hide]")
		return ExitError
	}
	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.Highlight(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		return HighlightResult{Highlighted: true, Selector: selector}, nil
	})
}

// ImagesResult is returned by the images command.
type ImagesResult struct {
	Images []cdp.ImageInfo `json:"images"`
	Count  int             `json:"count"`
}

func cmdImages(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		images, err := client.GetImages(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ImagesResult{Images: images, Count: len(images)}, nil
	})
}

// ScrollBottomResult is returned by the scrollbottom command.
type ScrollBottomResult struct {
	Scrolled bool `json:"scrolled"`
}

func cmdScrollBottom(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
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
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.ScrollToTop(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ScrollTopResult{Scrolled: true}, nil
	})
}

// FramesResult is returned by the frames command.
type FramesResult struct {
	Frames []cdp.FrameInfo `json:"frames"`
	Count  int             `json:"count"`
}

func cmdFrames(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		frames, err := client.GetFrames(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return FramesResult{Frames: frames, Count: len(frames)}, nil
	})
}

// EvalFrameResult is returned by the evalframe command.
type EvalFrameResult struct {
	FrameID string      `json:"frameId"`
	Type    string      `json:"type"`
	Value   interface{} `json:"value"`
}

func cmdEvalFrame(cfg *Config, frameID, expression string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		result, err := client.EvalInFrame(ctx, target.ID, frameID, expression)
		if err != nil {
			return nil, err
		}
		return EvalFrameResult{FrameID: frameID, Type: result.Type, Value: result.Value}, nil
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
		fmt.Fprintln(cfg.Stderr, "usage: cdp waitgone <selector> [--timeout <duration>]")
		return ExitError
	}
	selector := remaining[0]

	return withClientTarget(cfg, func(ctx context.Context, client *cdp.Client, target *cdp.TargetInfo) (interface{}, error) {
		err := client.WaitForGone(ctx, target.ID, selector, *timeout)
		if err != nil {
			return nil, err
		}
		return WaitGoneResult{Gone: true, Selector: selector}, nil
	})
}
