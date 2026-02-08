package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
)

func cmdRecord(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)

	outputFile := fs.String("output", "", "Write commands to file (default: stdout)")
	duration := fs.Duration("duration", 0, "Recording duration (0 = until interrupted)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	out := cfg.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
		defer f.Close()
		out = f
	}

	ctx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	connectCtx, connectCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer connectCancel()

	client, err := chrome.Connect(connectCtx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer client.Close()

	target, err := resolveTarget(connectCtx, client, cfg)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	events, err := client.RecordNavigations(ctx, target.ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	if !cfg.Quiet {
		fmt.Fprintln(cfg.Stderr, "Recording... (Ctrl+C to stop)")
	}

	fmt.Fprintf(out, "# hubcap recording %s\n", time.Now().Format(time.RFC3339))

	for event := range events {
		switch event.Type {
		case "navigate":
			fmt.Fprintf(out, "goto %s\n", event.URL)
		}
	}

	return ExitSuccess
}
