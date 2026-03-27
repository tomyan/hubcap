package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tomyan/hubcap/internal/chrome"
)

func cmdBridge(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("bridge", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	file := fs.String("file", "", "Load JS from file instead of inline argument")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	var script string
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(cfg.Stderr, "error: reading script file: %v\n", err)
			return ExitError
		}
		script = string(data)
	} else {
		remaining := fs.Args()
		if len(remaining) < 1 {
			fmt.Fprintln(cfg.Stderr, "usage: hubcap bridge <script> or hubcap bridge --file <file>")
			return ExitError
		}
		script = remaining[0]
	}

	ctx := context.Background()
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	client, err := chrome.Connect(ctx, cfg.Host, cfg.Port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: connecting to Chrome: %v\n", err)
		fmt.Fprintln(cfg.Stderr, "hint: run 'hubcap setup launch' to start Chrome")
		return ExitConnFailed
	}
	defer client.Close()

	target, err := resolveTarget(ctx, client, cfg)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	bridge, err := client.StartBridge(ctx, target.ID, script)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	defer bridge.Close()

	enc := json.NewEncoder(cfg.Stdout)
	for ev := range bridge.Events {
		enc.Encode(ev)
		if ev.Type == "closed" {
			return ExitSuccess
		}
	}

	return ExitSuccess
}
