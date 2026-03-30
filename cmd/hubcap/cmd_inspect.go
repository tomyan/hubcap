package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tomyan/hubcap/internal/chrome"
	sumi "github.com/tomyan/sumi/runtime/prelude"
	"github.com/tomyan/sumi-ui/components/console"
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

	// Signals.
	entries := sumi.New([]console.Entry{
		{Text: fmt.Sprintf("Connected to %s", title), Level: "verbose"},
	})
	prompt := sumi.New("")

	// Console component.
	comp := console.NewConsole(console.ConsoleProps{
		Entries: entries,
		Prompt:  prompt,
	})

	// App reference for Wake().
	var app *sumi.App

	// Key handling.
	comp.OnEvent = func(evt sumi.Event) {
		if evt.Kind == sumi.EventSignal {
			sumi.Quit()
			return
		}
		if evt.Ctrl && evt.Rune == 'c' {
			sumi.Quit()
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyEnter {
			expr := prompt.Get()
			if expr != "" {
				prompt.Set("")
				go evalAndAppend(client, ctx, target.ID, expr, entries, app)
			}
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyBackspace {
			val := []rune(prompt.Get())
			if len(val) > 0 {
				prompt.Set(string(val[:len(val)-1]))
			}
			return
		}
		if evt.Kind == sumi.EventKey && evt.Rune >= 32 && !evt.Ctrl && !evt.Alt {
			prompt.Update(func(cur string) string {
				return cur + string(evt.Rune)
			})
		}
	}

	// Stream CDP console messages.
	go func() {
		for msg := range messages {
			entries.Update(func(cur []console.Entry) []console.Entry {
				return append(cur, console.Entry{Text: msg.Text, Level: msg.Type})
			})
			if app != nil {
				app.Wake()
			}
		}
	}()

	sumi.RunWithOptions(comp, sumi.RunOptions{
		SetApp: func(a *sumi.App) { app = a },
	})
	return ExitSuccess
}

func evalAndAppend(client *chrome.Client, ctx context.Context, targetID, expr string, entries *sumi.Signal[[]console.Entry], app *sumi.App) {
	appendEntry(entries, "❯ "+expr, "info", app)
	result, err := client.Eval(ctx, targetID, expr)
	if err != nil {
		appendEntry(entries, "Error: "+err.Error(), "error", app)
		return
	}
	appendEntry(entries, formatEvalResult(result), "verbose", app)
}

func appendEntry(entries *sumi.Signal[[]console.Entry], text, level string, app *sumi.App) {
	entries.Update(func(cur []console.Entry) []console.Entry {
		return append(cur, console.Entry{Text: text, Level: level})
	})
	if app != nil {
		app.Wake()
	}
}

func formatEvalResult(result *chrome.EvalResult) string {
	if result == nil {
		return "undefined"
	}
	switch result.Type {
	case "undefined":
		return "undefined"
	case "string":
		if s, ok := result.Value.(string); ok {
			return fmt.Sprintf("%q", s)
		}
	}
	b, err := json.Marshal(result.Value)
	if err != nil {
		return fmt.Sprintf("%v", result.Value)
	}
	return string(b)
}
