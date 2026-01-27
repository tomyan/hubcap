package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testConfig() *Config {
	return &Config{
		Port:    9222,
		Host:    "localhost",
		Timeout: 5 * time.Second,
		Output:  "json",
		Quiet:   false,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRun_NoArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message in stderr, got: %s", stderr)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"unknown"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected 'unknown command' in stderr, got: %s", stderr)
	}
}

func TestRun_Help(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"-h"}, cfg)
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestRun_Version_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Port that won't have Chrome

	code := run([]string{"version"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "error:") {
		t.Errorf("expected error message in stderr, got: %s", stderr)
	}
}

func TestRun_Version_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"version"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Verify output is valid JSON
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, got: %s", err, stdout)
	}

	if result["browser"] == nil {
		t.Error("expected 'browser' field in output")
	}
}

func TestRun_Version_NDJSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"-output", "ndjson", "version"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, code)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	// NDJSON should not have indentation
	if strings.Contains(stdout, "\n  ") {
		t.Error("ndjson output should not be indented")
	}

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestRun_Version_CustomPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"-port", "9222", "version"}, cfg)
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestRun_InvalidOutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"-output", "invalid", "version"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d for invalid output format, got %d", ExitError, code)
	}
}
