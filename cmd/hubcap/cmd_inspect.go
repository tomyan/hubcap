package main

import (
	"context"
	"fmt"

	"github.com/tomyan/hubcap/internal/chrome"
	"github.com/tomyan/sumi/runtime/input"
	"github.com/tomyan/sumi/runtime/signal"
	"github.com/tomyan/sumi/runtime/tui"
	"github.com/tomyan/sumi-ui/components/scrollablelog"
)

func cmdInspect(cfg *Config, args []string) int {
	ctx := context.Background()

	client, err := chrome.Connect(ctx, cfg.Host, cfg.Port)
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

	// Get page title for status display.
	title, _ := client.GetTitle(ctx, target.ID)
	if title == "" {
		title = target.URL
	}

	messages, stopCapture, err := client.CaptureConsole(ctx, target.ID)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}
	defer stopCapture()

	// Signal holding the log entries, fed by CDP console events.
	entries := signal.New([]scrollablelog.LogEntry{
		{Text: fmt.Sprintf("Connected to %s", title), Level: "info"},
		{Text: "Waiting for console messages... (press q to quit)", Level: "verbose"},
	})

	comp := scrollablelog.NewScrollablelog(scrollablelog.ScrollablelogProps{
		Entries: entries,
	})

	// Wrap in an event handler for quit.
	comp.OnEvent = func(evt input.Event) {
		if evt.Kind == input.EventSignal {
			tui.Quit()
			return
		}
		if evt.Kind == input.EventKey && (evt.Rune == 'q' || (evt.Ctrl && evt.Rune == 'c')) {
			tui.Quit()
		}
	}

	// App reference for waking the render loop from the background goroutine.
	var app *tui.App

	// Background goroutine: read CDP console messages and append to signal.
	go func() {
		for msg := range messages {
			entries.Update(func(cur []scrollablelog.LogEntry) []scrollablelog.LogEntry {
				return append(cur, scrollablelog.LogEntry{
					Text:  msg.Text,
					Level: msg.Type,
				})
			})
			if app != nil {
				app.Wake()
			}
		}
	}()

	tui.RunWithOptions(comp, tui.RunOptions{
		SetApp: func(a *tui.App) { app = a },
	})
	return ExitSuccess
}
