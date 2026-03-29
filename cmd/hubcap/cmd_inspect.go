package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tomyan/hubcap/internal/chrome"
	"github.com/tomyan/sumi/runtime/input"
	"github.com/tomyan/sumi/runtime/layout"
	"github.com/tomyan/sumi/runtime/render"
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
	entries := signal.New([]scrollablelog.LogEntry{
		{Text: fmt.Sprintf("Connected to %s", title), Level: "verbose"},
	})
	prompt := signal.New("")

	// Build the log component and extract its tree.
	logComp := scrollablelog.NewScrollablelog(scrollablelog.ScrollablelogProps{
		Entries: entries,
	})
	// Make the log fill available height above the prompt.
	logComp.Tree.FlexGrow = 1

	// Prompt text node — updated reactively.
	promptNode := &layout.Input{
		Kind:    layout.KindText,
		Content: "> ",
		Style:   render.Style{Bold: true},
	}

	// Root layout: log fills space, prompt pinned at bottom.
	root := &layout.Input{
		Kind:      layout.KindBox,
		Direction: "column",
		CursorCol: -1,
		CursorRow: -1,
		Children: []*layout.Input{
			logComp.Tree,
			{
				Kind:      layout.KindBox,
				CursorCol: 2, // cursor after "> "
				CursorRow: 0,
				Children:  []*layout.Input{promptNode},
			},
		},
	}

	// Update prompt text and cursor position reactively.
	signal.Effect(func() {
		val := prompt.Get()
		promptNode.Content = "> " + val
		// Cursor at end of input.
		root.Children[1].CursorCol = 2 + len(val)
	})

	// App reference for Wake().
	var app *tui.App

	// Append a log entry and wake the render loop.
	appendEntry := func(text, level string) {
		entries.Update(func(cur []scrollablelog.LogEntry) []scrollablelog.LogEntry {
			return append(cur, scrollablelog.LogEntry{Text: text, Level: level})
		})
		if app != nil {
			app.Wake()
		}
	}

	// Evaluate JS expression and append result to log.
	evalExpr := func(expr string) {
		appendEntry("> "+expr, "info")
		result, err := client.Eval(ctx, target.ID, expr)
		if err != nil {
			appendEntry("Error: "+err.Error(), "error")
			return
		}
		text := formatEvalResult(result)
		appendEntry(text, "verbose")
	}

	comp := &tui.Component{
		Tree:        root,
		AfterLayout: logComp.AfterLayout,
		OnEvent: func(evt input.Event) {
			if evt.Kind == input.EventSignal {
				tui.Quit()
				return
			}
			if evt.Ctrl && evt.Rune == 'c' {
				tui.Quit()
				return
			}
			// Enter: evaluate expression.
			if evt.Kind == input.EventSpecial && evt.Special == input.KeyEnter {
				expr := prompt.Get()
				if expr != "" {
					prompt.Set("")
					go evalExpr(expr)
				}
				return
			}
			// Backspace.
			if evt.Kind == input.EventSpecial && evt.Special == input.KeyBackspace {
				val := prompt.Get()
				if len(val) > 0 {
					prompt.Set(val[:len(val)-1])
				}
				return
			}
			// Printable characters.
			if evt.Kind == input.EventKey && evt.Rune >= 32 && !evt.Ctrl && !evt.Alt {
				prompt.Update(func(cur string) string {
					return cur + string(evt.Rune)
				})
			}
		},
	}

	// Background goroutine: stream CDP console messages.
	go func() {
		for msg := range messages {
			appendEntry(msg.Text, msg.Type)
		}
	}()

	tui.RunWithOptions(comp, tui.RunOptions{
		SetApp: func(a *tui.App) { app = a },
	})
	return ExitSuccess
}

// formatEvalResult converts an eval result to a display string.
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
	// For objects, arrays, numbers — JSON encode the value.
	b, err := json.Marshal(result.Value)
	if err != nil {
		return fmt.Sprintf("%v", result.Value)
	}
	return string(b)
}
