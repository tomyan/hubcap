package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tomyan/hubcap/internal/chrome"
	"github.com/tomyan/hubcap/internal/inspector"
	"github.com/tomyan/sumi/runtime/edit"
	sumi "github.com/tomyan/sumi/runtime/prelude"
	"github.com/tomyan/sumi-ui/components/console"
)

// inspectSession manages the CDP connection lifecycle with auto-reconnect.
type inspectSession struct {
	host      string
	port      int
	targetCfg string // target selector from config

	mu             sync.Mutex
	client         *chrome.Client
	activeTargetID string

	// Signals updated on the main goroutine via app.Do.
	connected      *sumi.Signal[bool]
	entries        *sumi.Signal[[]console.Entry]
	pageTitle      *sumi.Signal[string]
	pageURL        *sumi.Signal[string]
	targetID       *sumi.Signal[string]
	browserVersion *sumi.Signal[string]
	tabs           *sumi.Signal[[]inspector.TabInfo]
	selectedIdx    *sumi.Signal[int]
	overlayVisible *sumi.Signal[bool]
	app            *sumi.App

	stopCapture func()
}

func (s *inspectSession) connect(ctx context.Context) error {
	client, err := chrome.Connect(ctx, s.host, s.port)
	if err != nil {
		return err
	}
	cfg := &Config{Host: s.host, Port: s.port, Target: s.targetCfg}
	target, err := resolveTarget(ctx, client, cfg)
	if err != nil {
		client.Close()
		return err
	}

	messages, stop, err := client.CaptureConsole(ctx, target.ID)
	if err != nil {
		client.Close()
		return err
	}

	// Fetch page info for the overlay.
	title, _ := client.GetTitle(ctx, target.ID)
	url, _ := client.GetURL(ctx, target.ID)
	version, _ := client.Version(ctx)
	browser := ""
	if version != nil {
		browser = version.Browser
	}

	s.mu.Lock()
	s.client = client
	s.activeTargetID = target.ID
	s.stopCapture = stop
	s.mu.Unlock()

	// Update signals on main goroutine.
	if s.app != nil {
		s.app.Do(func() {
			s.connected.Set(true)
			s.overlayVisible.Set(false)
			s.pageTitle.Set(title)
			s.pageURL.Set(url)
			s.targetID.Set(target.ID)
			s.browserVersion.Set(browser)
		})
	}

	// Populate tab list.
	go s.refreshTabs(ctx)

	// Stream console messages until the channel closes (disconnect).
	go func() {
		for msg := range messages {
			text, level := msg.Text, msg.Type
			if s.app != nil {
				s.app.Do(func() {
					s.entries.Update(func(cur []console.Entry) []console.Entry {
						return append(cur, console.Entry{Text: text, Level: level})
					})
				})
			}
		}
		// Channel closed — connection lost.
		s.handleDisconnect(ctx)
	}()

	return nil
}

func (s *inspectSession) reconnectLoop(ctx context.Context) {
	for {
		time.Sleep(2 * time.Second)
		if err := s.connect(ctx); err == nil {
			return // reconnected
		}
		// Still disconnected — keep trying.
	}
}

func (s *inspectSession) focusTab(ctx context.Context, tabTargetID string) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	client.Call(ctx, "Target.activateTarget", map[string]interface{}{
		"targetId": tabTargetID,
	})
}

func (s *inspectSession) newTab(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	client.NewTab(ctx, "")
}

func (s *inspectSession) refreshTabs(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	pages, err := client.AllPages(ctx)
	if err != nil {
		return
	}
	tabInfos := make([]inspector.TabInfo, len(pages))
	for i, p := range pages {
		tabInfos[i] = inspector.TabInfo{ID: p.ID, Title: p.Title, URL: p.URL}
	}
	if s.app != nil {
		s.app.Do(func() {
			s.tabs.Set(tabInfos)
			// Clamp selection if tabs list shrunk.
			if s.selectedIdx.Get() >= len(tabInfos) {
				s.selectedIdx.Set(max(0, len(tabInfos)-1))
			}
		})
	}
}

func (s *inspectSession) switchTarget(ctx context.Context, newTargetID string) {
	s.mu.Lock()
	client := s.client
	if s.stopCapture != nil {
		s.stopCapture()
		s.stopCapture = nil
	}
	s.mu.Unlock()
	if client == nil {
		return
	}

	messages, stop, err := client.CaptureConsole(ctx, newTargetID)
	if err != nil {
		return
	}

	title, _ := client.GetTitle(ctx, newTargetID)
	url, _ := client.GetURL(ctx, newTargetID)

	s.mu.Lock()
	s.activeTargetID = newTargetID
	s.stopCapture = stop
	s.mu.Unlock()

	if s.app != nil {
		s.app.Do(func() {
			s.targetID.Set(newTargetID)
			s.pageTitle.Set(title)
			s.pageURL.Set(url)
			s.entries.Set([]console.Entry{})
		})
	}

	// Stream console messages from new target.
	go func() {
		for msg := range messages {
			text, level := msg.Text, msg.Type
			if s.app != nil {
				s.app.Do(func() {
					s.entries.Update(func(cur []console.Entry) []console.Entry {
						return append(cur, console.Entry{Text: text, Level: level})
					})
				})
			}
		}
		s.handleDisconnect(ctx)
	}()
}

func (s *inspectSession) eval(ctx context.Context, expr string) (*chrome.EvalResult, error) {
	s.mu.Lock()
	client := s.client
	targetID := s.activeTargetID
	s.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected")
	}
	result, err := client.Eval(ctx, targetID, expr)
	if err != nil {
		// Session errors mean the target is gone — trigger disconnect.
		s.handleDisconnect(ctx)
	}
	return result, err
}

func (s *inspectSession) handleDisconnect(ctx context.Context) {
	s.mu.Lock()
	alreadyDisconnected := s.client == nil
	if s.stopCapture != nil {
		s.stopCapture()
		s.stopCapture = nil
	}
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
	s.mu.Unlock()

	if alreadyDisconnected {
		return
	}

	if s.app != nil {
		s.app.Do(func() {
			s.connected.Set(false)
			s.overlayVisible.Set(true)
		})
	}

	go s.reconnectLoop(ctx)
}

func (s *inspectSession) close() {
	s.mu.Lock()
	if s.stopCapture != nil {
		s.stopCapture()
	}
	if s.client != nil {
		s.client.Close()
	}
	s.mu.Unlock()
}

func (s *inspectSession) getTitle(ctx context.Context) string {
	s.mu.Lock()
	client := s.client
	targetID := s.activeTargetID
	s.mu.Unlock()
	if client == nil {
		return ""
	}
	title, _ := client.GetTitle(ctx, targetID)
	return title
}

func filteredTabs(allTabs []inspector.TabInfo, filter string) []inspector.TabInfo {
	if filter == "" {
		return allTabs
	}
	f := strings.ToLower(filter)
	var result []inspector.TabInfo
	for _, t := range allTabs {
		if strings.Contains(strings.ToLower(t.Title), f) || strings.Contains(strings.ToLower(t.URL), f) {
			result = append(result, t)
		}
	}
	return result
}

func cmdInspect(cfg *Config, args []string) int {
	ctx := context.Background()

	// Signals.
	entries := sumi.New([]console.Entry{})
	ed := &edit.State{}
	prompt := sumi.New("")
	cursor := sumi.New(0)
	connected := sumi.New(false)
	overlayVisible := sumi.New(false)
	filter := sumi.New("")
	pageTitle := sumi.New("")
	pageURL := sumi.New("")
	targetID := sumi.New("")
	browserVersion := sumi.New("")
	tabs := sumi.New([]inspector.TabInfo{})
	selectedIdx := sumi.New(0)

	sess := &inspectSession{
		host:           cfg.Host,
		port:           cfg.Port,
		targetCfg:      cfg.Target,
		connected:      connected,
		entries:        entries,
		pageTitle:      pageTitle,
		pageURL:        pageURL,
		targetID:       targetID,
		browserVersion: browserVersion,
		tabs:           tabs,
		selectedIdx:    selectedIdx,
		overlayVisible: overlayVisible,
	}

	// Initial connection.
	if err := sess.connect(ctx); err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitConnFailed
	}
	defer sess.close()

	syncPrompt := func() {
		prompt.Set(ed.Value)
		cursor.Set(ed.Cursor)
	}

	comp := inspector.NewInspector(inspector.InspectorProps{
		Entries:        entries,
		Prompt:         prompt,
		Cursor:         cursor,
		Connected:      connected,
		OverlayVisible: overlayVisible,
		PageTitle:      pageTitle,
		PageURL:        pageURL,
		TargetID:       targetID,
		BrowserVersion: browserVersion,
		Tabs:           tabs,
		SelectedIdx:    selectedIdx,
		Filter:         filter,
	})

	var app *sumi.App

	comp.OnEvent = func(evt sumi.Event) {
		if evt.Kind == sumi.EventSignal {
			sumi.Quit()
			return
		}
		if evt.Ctrl && evt.Rune == 'c' {
			sumi.Quit()
			return
		}

		// Ctrl+I (Tab) toggles connection overlay.
		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyTab {
			if !overlayVisible.Get() {
				filter.Set("")
				selectedIdx.Set(0)
				go sess.refreshTabs(ctx)
			}
			overlayVisible.Set(!overlayVisible.Get())
			return
		}

		// When overlay is visible, capture all input.
		if overlayVisible.Get() {
			if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyEscape {
				overlayVisible.Set(false)
				return
			}
			if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyUp {
				if selectedIdx.Get() > 0 {
					selectedIdx.Set(selectedIdx.Get() - 1)
				}
				return
			}
			if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyDown {
				ft := filteredTabs(tabs.Get(), filter.Get())
				if selectedIdx.Get() < len(ft)-1 {
					selectedIdx.Set(selectedIdx.Get() + 1)
				}
				return
			}
			if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyEnter {
				ft := filteredTabs(tabs.Get(), filter.Get())
				idx := selectedIdx.Get()
				if idx >= 0 && idx < len(ft) {
					overlayVisible.Set(false)
					go sess.switchTarget(ctx, ft[idx].ID)
				}
				return
			}
			// Shortcuts only when filter is empty.
			if filter.Get() == "" {
				if evt.Kind == sumi.EventKey && evt.Rune == 'f' {
					ft := filteredTabs(tabs.Get(), filter.Get())
					idx := selectedIdx.Get()
					if idx >= 0 && idx < len(ft) {
						go sess.focusTab(ctx, ft[idx].ID)
					}
					return
				}
				if evt.Kind == sumi.EventKey && evt.Rune == 'n' {
					go func() {
						sess.newTab(ctx)
						sess.refreshTabs(ctx)
					}()
					return
				}
			}
			// Backspace deletes filter char.
			if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyBackspace {
				f := filter.Get()
				if len(f) > 0 {
					filter.Set(f[:len(f)-1])
					selectedIdx.Set(0)
				}
				return
			}
			// Printable chars append to filter.
			if evt.Kind == sumi.EventKey && evt.Rune >= 32 {
				filter.Set(filter.Get() + string(evt.Rune))
				selectedIdx.Set(0)
				return
			}
			return
		}

		if evt.Kind == sumi.EventSpecial && evt.Special == sumi.KeyEnter {
			expr := ed.Submit()
			if expr != "" {
				syncPrompt()
				go evalAndAppendSession(sess, ctx, expr, entries, app)
			}
			return
		}

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

		if evt.Kind == sumi.EventPaste {
			for _, ch := range evt.PasteText {
				ed.Insert(ch)
			}
			syncPrompt()
			return
		}

		if evt.Kind == sumi.EventKey && evt.Rune >= 32 {
			ed.Insert(evt.Rune)
			syncPrompt()
		}
	}

	sumi.RunWithOptions(comp, sumi.RunOptions{
		SetApp: func(a *sumi.App) {
			app = a
			sess.app = a
			connected.Set(true)
		},
	})
	return ExitSuccess
}

func evalAndAppendSession(sess *inspectSession, ctx context.Context, expr string, entries *sumi.Signal[[]console.Entry], app *sumi.App) {
	appendEntry(entries, expr, "submitted", app)
	result, err := sess.eval(ctx, expr)
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
