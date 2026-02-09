package launcher

import (
	"fmt"
	"testing"
)

// mockCall records a single command invocation.
type mockCall struct {
	Name string
	Args []string
}

// mockRunner records commands and returns scripted results.
type mockRunner struct {
	calls   []mockCall
	results []mockResult
	callIdx int
}

type mockResult struct {
	output []byte
	err    error
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, mockCall{Name: name, Args: args})
	if m.callIdx < len(m.results) {
		r := m.results[m.callIdx]
		m.callIdx++
		return r.output, r.err
	}
	m.callIdx++
	return nil, nil
}

func (m *mockRunner) findCall(name string) *mockCall {
	for i := range m.calls {
		if m.calls[i].Name == name {
			return &m.calls[i]
		}
	}
	return nil
}

func (m *mockRunner) findCallArgs(name string, argSubstr string) *mockCall {
	for i := range m.calls {
		if m.calls[i].Name == name {
			for _, a := range m.calls[i].Args {
				if a == argSubstr || contains(a, argSubstr) {
					return &m.calls[i]
				}
			}
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Slice 1: quitChromeDarwin tests ---

func TestQuitChromeDarwin(t *testing.T) {
	t.Parallel()

	// Given
	// Call sequence: pgrep (running) → osascript (quit) → pgrep (gone)
	runner := &mockRunner{
		results: []mockResult{
			{nil, nil},                         // pgrep: Chrome is running
			{nil, nil},                         // osascript: quit succeeds
			{nil, fmt.Errorf("exit status 1")}, // pgrep: Chrome gone
		},
	}

	// When
	err := quitChromeDarwin(runner, 100)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	osascriptCall := runner.findCall("osascript")
	if osascriptCall == nil {
		t.Fatal("expected osascript to be called")
	}

	pgrepCall := runner.findCall("pgrep")
	if pgrepCall == nil {
		t.Fatal("expected pgrep to be called")
	}
}

func TestQuitChromeDarwin_NotRunning(t *testing.T) {
	t.Parallel()

	// Given
	// pgrep immediately returns "not found" — Chrome is not running
	runner := &mockRunner{
		results: []mockResult{
			{nil, fmt.Errorf("exit status 1")}, // pgrep: no Chrome found
		},
	}

	// When
	err := quitChromeDarwin(runner, 100)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// osascript should NOT have been called since Chrome wasn't running
	osascriptCall := runner.findCall("osascript")
	if osascriptCall != nil {
		t.Error("osascript should not be called when Chrome isn't running")
	}

	// Only one call: the initial pgrep check
	if len(runner.calls) != 1 {
		t.Errorf("expected 1 call (pgrep check), got %d: %+v", len(runner.calls), runner.calls)
	}
}

func TestQuitChromeDarwin_FallbackToKill(t *testing.T) {
	t.Parallel()

	// Given
	// osascript succeeds, but pgrep keeps finding Chrome, so we fall back to pkill
	results := []mockResult{
		{nil, nil},                  // osascript quit: success
	}
	// Add many pgrep "still running" results to exceed timeout
	for i := 0; i < 50; i++ {
		results = append(results, mockResult{[]byte("1234\n"), nil})
	}
	// pkill succeeds
	results = append(results, mockResult{nil, nil})

	runner := &mockRunner{results: results}

	// When — use tiny maxWaitMs so we quickly hit the fallback
	err := quitChromeDarwin(runner, 1)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkillCall := runner.findCall("pkill")
	if pkillCall == nil {
		t.Fatal("expected pkill to be called as fallback")
	}
}

// --- Slice 2: relaunchChromeDarwin tests ---

func TestRelaunchChromeDarwin(t *testing.T) {
	t.Parallel()

	// Given
	runner := &mockRunner{
		results: []mockResult{
			{nil, nil}, // open -a: success
		},
	}

	// When
	err := relaunchChromeDarwin(runner, 9222)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d: %+v", len(runner.calls), runner.calls)
	}

	call := runner.calls[0]
	if call.Name != "open" {
		t.Errorf("expected 'open', got %q", call.Name)
	}

	// Verify args: -a "Google Chrome" --args --remote-debugging-port=9222
	wantArgs := []string{"-a", "Google Chrome", "--args", "--remote-debugging-port=9222"}
	if len(call.Args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", call.Args, wantArgs)
	}
	for i, want := range wantArgs {
		if call.Args[i] != want {
			t.Errorf("arg[%d] = %q, want %q", i, call.Args[i], want)
		}
	}
}

func TestRelaunchChromeDarwin_CustomPort(t *testing.T) {
	t.Parallel()

	// Given
	runner := &mockRunner{
		results: []mockResult{
			{nil, nil}, // open -a: success
		},
	}

	// When
	err := relaunchChromeDarwin(runner, 9333)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := runner.calls[0]
	foundPort := false
	for _, arg := range call.Args {
		if arg == "--remote-debugging-port=9333" {
			foundPort = true
			break
		}
	}
	if !foundPort {
		t.Errorf("expected --remote-debugging-port=9333 in args, got %v", call.Args)
	}
}

// --- Slice 3: RelaunchUserChrome end-to-end tests ---

func TestRelaunchUserChrome_Darwin_FullSequence(t *testing.T) {
	t.Parallel()

	// Given
	// Full sequence: pgrep (running) → osascript (quit) → pgrep (gone) → open (relaunch)
	runner := &mockRunner{
		results: []mockResult{
			{nil, nil},                         // pgrep: Chrome is running
			{nil, nil},                         // osascript: quit succeeds
			{nil, fmt.Errorf("exit status 1")}, // pgrep: Chrome gone
			{nil, nil},                         // open -a: relaunch succeeds
		},
	}
	waitCalled := false

	// When
	opts := RelaunchOptions{
		Port:     9222,
		GOOS:     "darwin",
		Runner:   runner,
		WaitFunc: func() error { waitCalled = true; return nil },
	}
	err := RelaunchUserChrome(opts)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify quit happened (osascript was called)
	if runner.findCall("osascript") == nil {
		t.Error("expected osascript to be called for quit")
	}

	// Verify relaunch happened (open was called)
	openCall := runner.findCall("open")
	if openCall == nil {
		t.Fatal("expected open to be called for relaunch")
	}

	// Verify wait-for-port was called
	if !waitCalled {
		t.Error("expected WaitFunc to be called")
	}
}

func TestRelaunchUserChrome_ChromeNotRunning_JustLaunches(t *testing.T) {
	t.Parallel()

	// Given
	// pgrep says Chrome not running → skip quit → open (relaunch)
	runner := &mockRunner{
		results: []mockResult{
			{nil, fmt.Errorf("exit status 1")}, // pgrep: no Chrome
			{nil, nil},                         // open -a: relaunch succeeds
		},
	}
	waitCalled := false

	// When
	opts := RelaunchOptions{
		Port:     9222,
		GOOS:     "darwin",
		Runner:   runner,
		WaitFunc: func() error { waitCalled = true; return nil },
	}
	err := RelaunchUserChrome(opts)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// osascript should NOT have been called
	if runner.findCall("osascript") != nil {
		t.Error("osascript should not be called when Chrome wasn't running")
	}

	// open should have been called
	if runner.findCall("open") == nil {
		t.Fatal("expected open to be called for relaunch")
	}

	if !waitCalled {
		t.Error("expected WaitFunc to be called")
	}
}

func TestRelaunchUserChrome_UnsupportedOS(t *testing.T) {
	t.Parallel()

	// Given
	runner := &mockRunner{}

	// When
	opts := RelaunchOptions{
		Port:     9222,
		GOOS:     "windows",
		Runner:   runner,
		WaitFunc: func() error { return nil },
	}
	err := RelaunchUserChrome(opts)

	// Then
	if err == nil {
		t.Fatal("expected error for unsupported OS")
	}
}
