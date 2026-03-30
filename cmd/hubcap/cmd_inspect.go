package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
	"github.com/tomyan/hubcap/internal/inspector"
	"github.com/tomyan/sumi/runtime/edit"
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
	entries := sumi.New([]console.Entry{})
	ed := &edit.State{}
	prompt := sumi.New("")
	cursor := sumi.New(0)
	connected := sumi.New(true)

	// syncPrompt updates the prompt signal and cursor from the edit state.
	syncPrompt := func() {
		prompt.Set(ed.Value)
		cursor.Set(ed.Cursor)
	}

	// Inspector component (topbar + console panel).
	comp := inspector.NewInspector(inspector.InspectorProps{
		Entries:   entries,
		Prompt:    prompt,
		Cursor:    cursor,
		Connected: connected,
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

		// Submit.
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyEnter {
			expr := ed.Submit()
			if expr != "" {
				syncPrompt()
				go evalAndAppend(client, ctx, target.ID, expr, entries, app)
			}
			return
		}

		// History.
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyUp {
			ed.HistoryUp()
			syncPrompt()
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyDown {
			ed.HistoryDown()
			syncPrompt()
			return
		}

		// Navigation.
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyLeft {
			ed.Left()
			syncPrompt()
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyRight {
			ed.Right()
			syncPrompt()
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyHome {
			ed.Home()
			syncPrompt()
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyEnd {
			ed.End()
			syncPrompt()
			return
		}

		// Deletion.
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyBackspace {
			ed.Backspace()
			syncPrompt()
			return
		}
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyDelete {
			ed.Delete()
			syncPrompt()
			return
		}

		// Readline shortcuts.
		if evt.Ctrl {
			switch evt.Rune {
			case 'a':
				ed.Home()
			case 'e':
				ed.End()
			case 'k':
				ed.KillToEnd()
			case 'u':
				ed.KillToStart()
			case 'w':
				ed.KillWord()
			case 'y':
				ed.Yank()
			case 't':
				ed.TransposeChars()
			case '_':
				ed.Undo()
			default:
				return
			}
			syncPrompt()
			return
		}

		// Alt key combinations.
		if evt.Alt {
			if evt.Kind == sumi.EventKey {
				switch evt.Rune {
				case 'b':
					ed.WordLeft()
				case 'f':
					ed.WordRight()
				case 'd':
					ed.KillWordForward()
				case 'y':
					ed.YankPop()
				case 'u':
					ed.UppercaseWord()
				case 'l':
					ed.LowercaseWord()
				case 'c':
					ed.CapitalizeWord()
				default:
					return
				}
				syncPrompt()
			}
			return
		}

		// Paste.
		if evt.Kind == sumi.EventPaste {
			for _, ch := range evt.PasteText {
				ed.Insert(ch)
			}
			syncPrompt()
			return
		}

		// Printable characters.
		if evt.Kind == sumi.EventKey && evt.Rune >= 32 {
			ed.Insert(evt.Rune)
			syncPrompt()
		}
	}

	// Stream CDP console messages — mutations via app.Do() to stay on main goroutine.
	go func() {
		for msg := range messages {
			text, level := msg.Text, msg.Type
			if app != nil {
				app.Do(func() {
					entries.Update(func(cur []console.Entry) []console.Entry {
						return append(cur, console.Entry{Text: text, Level: level})
					})
				})
			}
		}
	}()

	sumi.RunWithOptions(comp, sumi.RunOptions{
		SetApp: func(a *sumi.App) { app = a },
	})
	return ExitSuccess
}

func evalAndAppend(client *chrome.Client, ctx context.Context, targetID, expr string, entries *sumi.Signal[[]console.Entry], app *sumi.App) {
	appendEntry(entries, expr, "submitted", app)
	result, err := client.Eval(ctx, targetID, expr)
	// Small yield to let any console output from the expression (e.g. console.log)
	// arrive via the CDP event stream before we append the return value.
	time.Sleep(50 * time.Millisecond)
	if err != nil {
		appendEntry(entries, "Error: "+err.Error(), "error", app)
		return
	}
	appendEntry(entries, formatEvalResult(result), "result", app)
}

func appendEntry(entries *sumi.Signal[[]console.Entry], text, level string, app *sumi.App) {
	if app != nil {
		app.Do(func() {
			entries.Update(func(cur []console.Entry) []console.Entry {
				return append(cur, console.Entry{Text: text, Level: level})
			})
		})
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
