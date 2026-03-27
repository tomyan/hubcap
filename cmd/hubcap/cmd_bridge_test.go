package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRun_Bridge_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"bridge"}, cfg)
	if code != ExitError {
		t.Errorf("expected ExitError, got %d", code)
	}
}

func TestRun_Bridge_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 19999
	code := run([]string{"bridge", "send(1)"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected ExitConnFailed, got %d", code)
	}
}

func TestRun_Bridge_SendMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page first
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "--wait", "data:text/html,<html><body>bridge</body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %s", cfg.Stderr.(*bytes.Buffer).String())
	}

	// Run bridge with a script that sends a message and exits
	cfg = testConfig()
	cfg.Timeout = 5 * time.Second
	code = run([]string{"--target", tabID, "bridge", `send({hello: "world"})`}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected ExitSuccess, got %d, stderr: %s", code, stderr)
	}

	// Parse LDJSON output
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (ready, message, closed), got %d: %s", len(lines), stdout)
	}

	// First line: ready
	var ready map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &ready); err != nil {
		t.Fatalf("failed to parse ready line: %v", err)
	}
	if ready["type"] != "ready" {
		t.Errorf("expected type 'ready', got %v", ready["type"])
	}

	// Second line: message
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &msg); err != nil {
		t.Fatalf("failed to parse message line: %v", err)
	}
	if msg["type"] != "message" {
		t.Errorf("expected type 'message', got %v", msg["type"])
	}
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map data, got %T", msg["data"])
	}
	if data["hello"] != "world" {
		t.Errorf("expected hello='world', got %v", data["hello"])
	}

	// Third line: closed
	var closed map[string]interface{}
	if err := json.Unmarshal([]byte(lines[2]), &closed); err != nil {
		t.Fatalf("failed to parse closed line: %v", err)
	}
	if closed["type"] != "closed" {
		t.Errorf("expected type 'closed', got %v", closed["type"])
	}
}

func TestRun_Bridge_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "--wait", "data:text/html,<html><body>bridge</body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %s", cfg.Stderr.(*bytes.Buffer).String())
	}

	// Provide stdin with a message followed by close
	stdinData := `{"data":{"n":7}}` + "\n" + `{"type":"close"}` + "\n"

	// Run bridge with a script that echoes messages
	cfg = testConfig()
	cfg.Timeout = 5 * time.Second
	cfg.Stdin = strings.NewReader(stdinData)
	code = run([]string{"--target", tabID, "bridge", `
		for await (const msg of messages) {
			send({doubled: msg.n * 2});
			break;
		}
	`}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected ExitSuccess, got %d, stderr: %s", code, stderr)
	}

	// Parse output — expect ready, message (doubled), closed
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %s", len(lines), stdout)
	}

	// Find the message line
	var found bool
	for _, line := range lines {
		var ev map[string]interface{}
		json.Unmarshal([]byte(line), &ev)
		if ev["type"] == "message" {
			data := ev["data"].(map[string]interface{})
			if data["doubled"] != float64(14) {
				t.Errorf("expected doubled=14, got %v", data["doubled"])
			}
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no message event found in output: %s", stdout)
	}
}
