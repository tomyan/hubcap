package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tomyan/hubcap/internal/chrome"
)

// AssertResult is returned by the assert command on success.
type AssertResult struct {
	Passed    bool   `json:"passed"`
	Assertion string `json:"assertion"`
}

func cmdAssert(cfg *Config, args []string) int {
	if len(args) < 1 {
		printAssertUsage(cfg)
		return ExitError
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "text":
		if len(rest) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap assert text <selector> <expected>")
			return ExitError
		}
		return assertText(cfg, rest[0], rest[1])
	case "title":
		if len(rest) < 1 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap assert title <expected>")
			return ExitError
		}
		return assertTitle(cfg, rest[0])
	case "url":
		if len(rest) < 1 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap assert url <substring>")
			return ExitError
		}
		return assertURL(cfg, rest[0])
	case "exists":
		if len(rest) < 1 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap assert exists <selector>")
			return ExitError
		}
		return assertExists(cfg, rest[0])
	case "visible":
		if len(rest) < 1 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap assert visible <selector>")
			return ExitError
		}
		return assertVisible(cfg, rest[0])
	case "count":
		if len(rest) < 2 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap assert count <selector> <expected-count>")
			return ExitError
		}
		return assertCount(cfg, rest[0], rest[1])
	default:
		fmt.Fprintf(cfg.Stderr, "unknown assertion: %s\n", sub)
		printAssertUsage(cfg)
		return ExitError
	}
}

func printAssertUsage(cfg *Config) {
	fmt.Fprintln(cfg.Stderr, "usage: hubcap assert <assertion> [args...]")
	fmt.Fprintln(cfg.Stderr)
	fmt.Fprintln(cfg.Stderr, "assertions:")
	fmt.Fprintln(cfg.Stderr, "  text <selector> <expected>    Assert element text equals expected")
	fmt.Fprintln(cfg.Stderr, "  title <expected>              Assert page title equals expected")
	fmt.Fprintln(cfg.Stderr, "  url <substring>               Assert URL contains substring")
	fmt.Fprintln(cfg.Stderr, "  exists <selector>             Assert element exists")
	fmt.Fprintln(cfg.Stderr, "  visible <selector>            Assert element is visible")
	fmt.Fprintln(cfg.Stderr, "  count <selector> <n>          Assert element count equals n")
}

func assertText(cfg *Config, selector, expected string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		actual, err := client.GetText(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		if actual != expected {
			return nil, fmt.Errorf("text mismatch for %q: got %q, want %q", selector, actual, expected)
		}
		return AssertResult{Passed: true, Assertion: fmt.Sprintf("text(%s) == %q", selector, expected)}, nil
	})
}

func assertTitle(cfg *Config, expected string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		actual, err := client.GetTitle(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		if actual != expected {
			return nil, fmt.Errorf("title mismatch: got %q, want %q", actual, expected)
		}
		return AssertResult{Passed: true, Assertion: fmt.Sprintf("title == %q", expected)}, nil
	})
}

func assertURL(cfg *Config, substring string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		actual, err := client.GetURL(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(actual, substring) {
			return nil, fmt.Errorf("URL %q does not contain %q", actual, substring)
		}
		return AssertResult{Passed: true, Assertion: fmt.Sprintf("url contains %q", substring)}, nil
	})
}

func assertExists(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		exists, err := client.Exists(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("element not found: %s", selector)
		}
		return AssertResult{Passed: true, Assertion: fmt.Sprintf("exists(%s)", selector)}, nil
	})
}

func assertVisible(cfg *Config, selector string) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		visible, err := client.IsVisible(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		if !visible {
			return nil, fmt.Errorf("element not visible: %s", selector)
		}
		return AssertResult{Passed: true, Assertion: fmt.Sprintf("visible(%s)", selector)}, nil
	})
}

func assertCount(cfg *Config, selector, expectedStr string) int {
	expected, err := strconv.Atoi(expectedStr)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "invalid count: %s\n", expectedStr)
		return ExitError
	}

	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		actual, err := client.CountElements(ctx, target.ID, selector)
		if err != nil {
			return nil, err
		}
		if actual != expected {
			return nil, fmt.Errorf("count mismatch for %q: got %d, want %d", selector, actual, expected)
		}
		return AssertResult{Passed: true, Assertion: fmt.Sprintf("count(%s) == %d", selector, expected)}, nil
	})
}
