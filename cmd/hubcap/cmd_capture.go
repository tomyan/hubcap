package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/tomyan/hubcap/internal/chrome"
)

// ElementScreenshotResult contains metadata about an element screenshot.
type ElementScreenshotResult struct {
	Format   string              `json:"format"`
	Size     int                 `json:"size"`
	Selector string              `json:"selector,omitempty"`
	Bounds   *chrome.BoundingBox `json:"bounds,omitempty"`
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
		fmt.Fprintln(cfg.Stderr, "usage: hubcap screenshot --output <file> [--format png|jpeg|webp] [--quality 0-100] [--selector <css>]")
		fmt.Fprintln(cfg.Stderr, "       hubcap screenshot --base64 [--format png|jpeg|webp] [--quality 0-100]")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		opts := chrome.ScreenshotOptions{
			Format:  *format,
			Quality: *quality,
		}

		var data []byte
		var bounds *chrome.BoundingBox
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

		return chrome.ScreenshotResult{
			Format: *format,
			Size:   len(data),
		}, nil
	})
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
		fmt.Fprintln(cfg.Stderr, "usage: hubcap pdf --output <file> [--landscape] [--background]")
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		data, err := client.PrintToPDF(ctx, target.ID, chrome.PDFOptions{
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
