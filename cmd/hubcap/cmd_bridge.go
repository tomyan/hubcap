package main

import (
	"bufio"
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

	// Read stdin in a goroutine and forward messages to the bridge
	stdinDone := make(chan struct{})
	go func() {
		defer close(stdinDone)
		if cfg.Stdin == nil {
			return
		}
		scanner := bufio.NewScanner(cfg.Stdin)
		for scanner.Scan() {
			line := scanner.Bytes()
			var envelope struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(line, &envelope); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: invalid JSON on stdin: %v\n", err)
				continue
			}

			if envelope.Type == "close" {
				if err := bridge.CloseIterator(ctx); err != nil {
					fmt.Fprintf(cfg.Stderr, "error: closing bridge iterator: %v\n", err)
				}
				return
			}

			var data interface{}
			if err := json.Unmarshal(envelope.Data, &data); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: invalid data in message: %v\n", err)
				continue
			}

			if err := bridge.Send(ctx, data); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: sending to bridge: %v\n", err)
				return
			}
		}
		// stdin EOF — signal JS to close gracefully
		if err := bridge.CloseIterator(ctx); err != nil {
			fmt.Fprintf(cfg.Stderr, "error: closing bridge iterator: %v\n", err)
		}
	}()

	enc := json.NewEncoder(cfg.Stdout)
	for ev := range bridge.Events {
		enc.Encode(ev)
		if ev.Type == "closed" {
			return ExitSuccess
		}
	}

	return ExitSuccess
}
