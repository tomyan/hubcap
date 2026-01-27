package main

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestRun_Tabs_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"tabs"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Verify output is valid JSON array
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result []interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON array: %v, got: %s", err, stdout)
	}
}

func TestRun_Tabs_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"tabs"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Tabs_OutputContainsPageInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"tabs"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, code)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var tabs []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &tabs); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Each tab should have id, type, title, url
	for i, tab := range tabs {
		if tab["id"] == nil {
			t.Errorf("tab %d missing 'id' field", i)
		}
		if tab["type"] == nil {
			t.Errorf("tab %d missing 'type' field", i)
		}
	}
}

func TestRun_Goto_MissingURL(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"goto"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Goto_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"goto", "https://example.com"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["url"] == nil {
		t.Error("expected 'url' field in output")
	}
}

func TestRun_Goto_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"goto", "https://example.com"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Screenshot_MissingOutput(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"screenshot"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Screenshot_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	tmpFile := t.TempDir() + "/screenshot.png"

	code := run([]string{"screenshot", "--output", tmpFile}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Verify JSON output
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["format"] == nil {
		t.Error("expected 'format' field in output")
	}
	if result["size"] == nil {
		t.Error("expected 'size' field in output")
	}

	// Verify file was written with valid PNG data
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read screenshot file: %v", err)
	}

	// PNG magic bytes
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(data) < 8 {
		t.Fatal("screenshot file too small")
	}
	for i, b := range pngMagic {
		if data[i] != b {
			t.Fatalf("file is not valid PNG: byte %d is %x, expected %x", i, data[i], b)
		}
	}
}

func TestRun_Screenshot_JPEG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	tmpFile := t.TempDir() + "/screenshot.jpg"

	code := run([]string{"screenshot", "--output", tmpFile, "--format", "jpeg", "--quality", "90"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result["format"] != "jpeg" {
		t.Errorf("expected format 'jpeg', got %v", result["format"])
	}

	// Verify file was written with valid JPEG data
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read screenshot file: %v", err)
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatalf("file is not valid JPEG: got %x %x", data[0], data[1])
	}
}

func TestRun_Screenshot_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"screenshot", "--output", "/tmp/test.png"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Eval_MissingExpression(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"eval"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Eval_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"eval", "1 + 2"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["value"] != float64(3) {
		t.Errorf("expected value 3, got %v", result["value"])
	}
}

func TestRun_Eval_String(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	code := run([]string{"eval", "'hello'"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["value"] != "hello" {
		t.Errorf("expected value 'hello', got %v", result["value"])
	}
}

func TestRun_Eval_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"eval", "1 + 2"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Query_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"query"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Query_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// First navigate to a page
	run([]string{"goto", "https://example.com"}, cfg)

	// Reset stdout
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"query", "body"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["tagName"] != "BODY" {
		t.Errorf("expected tagName 'BODY', got %v", result["tagName"])
	}
}

func TestRun_Query_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"query", "body"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Click_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"click"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Click_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second // Longer timeout for click+navigation

	// First navigate to a page with clickable element
	run([]string{"goto", "https://example.com"}, cfg)

	// Reset buffers
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Click on body (non-navigating element) instead of link
	code := run([]string{"click", "body"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Click returns simple success message
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["clicked"] != true {
		t.Errorf("expected clicked: true, got %v", result["clicked"])
	}
}

func TestRun_Click_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"click", "body"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Fill_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"fill"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Fill_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate to blank page to reset state
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)

	// Create a page with an input
	run([]string{"eval", `document.body.innerHTML = '<input id="test-input" type="text" />'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	// Reset buffers
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"fill", "#test-input", "test value"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["filled"] != true {
		t.Errorf("expected filled: true, got %v", result["filled"])
	}
}

func TestRun_Fill_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1

	code := run([]string{"fill", "#input", "text"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_HTML_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"html"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_HTML_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to blank page to reset state
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)

	// Create a test element
	run([]string{"eval", `document.body.innerHTML = '<div id="test">Content</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"html", "#test"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	html, ok := result["html"].(string)
	if !ok || !strings.Contains(html, "Content") {
		t.Errorf("expected HTML containing 'Content', got %v", result["html"])
	}
}

func TestRun_Wait_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"wait"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Wait_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to blank page to reset state
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)

	// Create element that exists
	run([]string{"eval", `document.body.innerHTML = '<div id="exists">Test</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"wait", "#exists"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["found"] != true {
		t.Errorf("expected found: true, got %v", result["found"])
	}
}

func TestRun_Type_MissingText(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"type"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Type_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to blank page to reset state
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)

	// Create an input field and focus it
	run([]string{"eval", `document.body.innerHTML = '<input id="test-input" type="text" />'; document.querySelector('#test-input').focus();`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"type", "hello"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["typed"] != true {
		t.Errorf("expected typed: true, got %v", result["typed"])
	}
	if result["text"] != "hello" {
		t.Errorf("expected text: hello, got %v", result["text"])
	}
}

func TestRun_Type_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"type", "test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Console_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Run console with short duration - should exit cleanly after timeout
	code := run([]string{"console", "--duration", "100ms"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}
}

func TestRun_Console_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"console", "--duration", "100ms"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}
