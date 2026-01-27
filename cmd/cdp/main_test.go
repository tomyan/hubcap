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

func TestRun_Cookies_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to a page first
	run([]string{"goto", "https://example.com"}, cfg)
	time.Sleep(100 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"cookies"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Output should be valid JSON (array of cookies)
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var cookies []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &cookies); err != nil {
		t.Errorf("output is not valid JSON array: %v, got: %s", err, stdout)
	}
}

func TestRun_Cookies_Set(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to a page first
	run([]string{"goto", "https://example.com"}, cfg)
	time.Sleep(100 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"cookies", "--set", "test_cookie=test_value", "--domain", "example.com"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["set"] != true {
		t.Errorf("expected set: true, got %v", result["set"])
	}
	if result["name"] != "test_cookie" {
		t.Errorf("expected name: test_cookie, got %v", result["name"])
	}
}

func TestRun_Cookies_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to a page first
	run([]string{"goto", "https://example.com"}, cfg)
	time.Sleep(100 * time.Millisecond)

	// Set a cookie first
	run([]string{"cookies", "--set", "delete_me=value", "--domain", "example.com"}, cfg)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Delete the cookie
	code := run([]string{"cookies", "--delete", "delete_me", "--domain", "example.com"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["deleted"] != true {
		t.Errorf("expected deleted: true, got %v", result["deleted"])
	}
}

func TestRun_Cookies_Clear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to a page first
	run([]string{"goto", "https://example.com"}, cfg)
	time.Sleep(100 * time.Millisecond)

	// Set some cookies
	run([]string{"cookies", "--set", "clear1=val1", "--domain", "example.com"}, cfg)
	run([]string{"cookies", "--set", "clear2=val2", "--domain", "example.com"}, cfg)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Clear all cookies
	code := run([]string{"cookies", "--clear"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["cleared"] != true {
		t.Errorf("expected cleared: true, got %v", result["cleared"])
	}
}

func TestRun_Cookies_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"cookies"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_PDF_MissingOutput(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"pdf"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_PDF_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to a page first
	run([]string{"goto", "https://example.com"}, cfg)
	time.Sleep(100 * time.Millisecond)

	// Create a temp file for output
	tmpfile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"pdf", "--output", tmpfile.Name()}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Verify PDF was created and has PDF magic bytes
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Error("output file is not a valid PDF")
	}
}

func TestRun_PDF_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"pdf", "--output", "/tmp/test.pdf"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Focus_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"focus"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Focus_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()

	// Navigate to blank and create input
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<input id="focus-input" type="text" />'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"focus", "#focus-input"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["focused"] != true {
		t.Errorf("expected focused: true, got %v", result["focused"])
	}
}

func TestRun_Focus_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"focus", "#test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Network_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// First navigate to about:blank
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate to blank: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Run network capture with short duration
	code = run([]string{"network", "--duration", "500ms"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}
}

func TestRun_Network_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"network", "--duration", "100ms"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Press_MissingKey(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"press"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Press_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to blank page and create an input
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	code = run([]string{"eval", `document.body.innerHTML = '<input id="press-input" type="text" />'`}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create input: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	code = run([]string{"focus", "#press-input"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to focus: %d", code)
	}

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"press", "Enter"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["pressed"] != true {
		t.Errorf("expected pressed: true, got %v", result["pressed"])
	}
	if result["key"] != "Enter" {
		t.Errorf("expected key: Enter, got %v", result["key"])
	}
}

func TestRun_Press_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"press", "Enter"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Hover_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"hover"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Hover_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to blank page and create a button
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	code = run([]string{"eval", `document.body.innerHTML = '<button id="hover-btn" style="width:100px;height:50px;">Hover</button>'`}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create button: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"hover", "#hover-btn"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["hovered"] != true {
		t.Errorf("expected hovered: true, got %v", result["hovered"])
	}
}

func TestRun_Hover_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"hover", "#test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Attr_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"attr"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	cfg = testConfig()
	code = run([]string{"attr", "#test"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Attr_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to blank page and create a link
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	code = run([]string{"eval", `document.body.innerHTML = '<a id="link" href="https://test.com">Test</a>'`}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create link: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"attr", "#link", "href"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["value"] != "https://test.com" {
		t.Errorf("expected value 'https://test.com', got %v", result["value"])
	}
}

func TestRun_Attr_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"attr", "#test", "href"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Reload_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to a page first
	code := run([]string{"goto", "https://example.com"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(100 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"reload"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["reloaded"] != true {
		t.Errorf("expected reloaded: true, got %v", result["reloaded"])
	}
}

func TestRun_Reload_BypassCache(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to a page first
	code := run([]string{"goto", "https://example.com"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(100 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"reload", "--bypass-cache"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["ignoreCache"] != true {
		t.Errorf("expected ignoreCache: true, got %v", result["ignoreCache"])
	}
}

func TestRun_Reload_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"reload"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Back_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate to first page
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate to blank: %d", code)
	}
	time.Sleep(100 * time.Millisecond)

	// Navigate to second page
	code = run([]string{"goto", "https://example.com"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate to example: %d", code)
	}
	time.Sleep(100 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Go back
	code = run([]string{"back"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["success"] != true {
		t.Errorf("expected success: true, got %v", result["success"])
	}
}

func TestRun_Back_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"back"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Forward_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"forward"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Title_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to blank page and set title
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	// Set a title via eval
	code = run([]string{"eval", `document.title = "Test Title"`}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to set title: %d", code)
	}

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"title"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["title"] != "Test Title" {
		t.Errorf("expected title 'Test Title', got %v", result["title"])
	}
}

func TestRun_Title_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"title"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_URL_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to about:blank
	code := run([]string{"goto", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"url"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	url := result["url"].(string)
	if url != "about:blank" {
		t.Errorf("expected URL 'about:blank', got %s", url)
	}
}

func TestRun_URL_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"url"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_New_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"new"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["targetId"] == "" {
		t.Error("expected non-empty targetId")
	}

	// Clean up - close the new tab
	cfg.Stdout = &bytes.Buffer{}
	run([]string{"close"}, cfg)
}

func TestRun_New_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"new"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_TargetFlag_ByIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Create a new tab
	code := run([]string{"new", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create new tab: %d", code)
	}
	time.Sleep(100 * time.Millisecond)

	// Get the list of tabs
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"tabs"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to list tabs: %d", code)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var tabs []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &tabs); err != nil {
		t.Fatalf("failed to parse tabs: %v", err)
	}

	if len(tabs) < 2 {
		t.Skip("need at least 2 tabs for this test")
	}

	// Set a unique title on each tab using --target flag
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"--target", "0", "eval", "document.title = 'Tab Zero'"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to eval on tab 0: %d, stderr: %s", code, stderr)
	}

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"--target", "1", "eval", "document.title = 'Tab One'"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to eval on tab 1: %d, stderr: %s", code, stderr)
	}

	// Verify titles
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"--target", "0", "title"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to get title for tab 0: %d", code)
	}

	var result map[string]interface{}
	stdout = cfg.Stdout.(*bytes.Buffer).String()
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["title"] != "Tab Zero" {
		t.Errorf("expected title 'Tab Zero', got %v", result["title"])
	}

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"--target", "1", "title"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to get title for tab 1: %d", code)
	}

	stdout = cfg.Stdout.(*bytes.Buffer).String()
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["title"] != "Tab One" {
		t.Errorf("expected title 'Tab One', got %v", result["title"])
	}

	// Clean up - close the second tab
	cfg.Stdout = &bytes.Buffer{}
	run([]string{"--target", "1", "close"}, cfg)
}

func TestRun_TargetFlag_ByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Create a new tab
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code := run([]string{"new", "about:blank"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create new tab: %d", code)
	}

	var newResult map[string]interface{}
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	if err := json.Unmarshal([]byte(stdout), &newResult); err != nil {
		t.Fatalf("failed to parse new tab result: %v", err)
	}

	targetID := newResult["targetId"].(string)
	time.Sleep(100 * time.Millisecond)

	// Use the target ID to set a title
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"--target", targetID, "eval", "document.title = 'By ID'"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to eval with target ID: %d, stderr: %s", code, stderr)
	}

	// Verify title
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}
	code = run([]string{"--target", targetID, "title"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to get title: %d", code)
	}

	var result map[string]interface{}
	stdout = cfg.Stdout.(*bytes.Buffer).String()
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["title"] != "By ID" {
		t.Errorf("expected title 'By ID', got %v", result["title"])
	}

	// Clean up
	cfg.Stdout = &bytes.Buffer{}
	run([]string{"--target", targetID, "close"}, cfg)
}

func TestRun_TargetFlag_InvalidIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Try to use an invalid index
	code := run([]string{"--target", "999", "title"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d for invalid target, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "invalid target") {
		t.Errorf("expected 'invalid target' error, got: %s", stderr)
	}
}

func TestRun_Emulate_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"emulate", "iPhone 12"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["device"] != "iPhone 12" {
		t.Errorf("expected device 'iPhone 12', got %v", result["device"])
	}
	if result["width"] == nil {
		t.Error("expected width field in output")
	}
	if result["height"] == nil {
		t.Error("expected height field in output")
	}
}

func TestRun_Emulate_MissingDevice(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"emulate"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Emulate_InvalidDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"emulate", "NonexistentDevice"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d for invalid device, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "unknown device") {
		t.Errorf("expected 'unknown device' error, got: %s", stderr)
	}
}

func TestRun_Emulate_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"emulate", "iPhone 12"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_UserAgent_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"useragent", "TestBot/1.0"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["userAgent"] != "TestBot/1.0" {
		t.Errorf("expected userAgent 'TestBot/1.0', got %v", result["userAgent"])
	}
}

func TestRun_UserAgent_MissingArg(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"useragent"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_UserAgent_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"useragent", "TestBot/1.0"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Geolocation_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"geolocation", "37.7749", "-122.4194"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["latitude"] != 37.7749 {
		t.Errorf("expected latitude 37.7749, got %v", result["latitude"])
	}
	if result["longitude"] != -122.4194 {
		t.Errorf("expected longitude -122.4194, got %v", result["longitude"])
	}
}

func TestRun_Geolocation_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"geolocation"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	cfg = testConfig()
	code = run([]string{"geolocation", "37.7749"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Geolocation_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"geolocation", "37.7749", "-122.4194"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Offline_Enable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"offline", "true"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["offline"] != true {
		t.Errorf("expected offline true, got %v", result["offline"])
	}

	// Disable offline mode after test
	cfg.Stdout = &bytes.Buffer{}
	run([]string{"offline", "false"}, cfg)
}

func TestRun_Offline_Disable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"offline", "false"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["offline"] != false {
		t.Errorf("expected offline false, got %v", result["offline"])
	}
}

func TestRun_Offline_MissingArg(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"offline"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Offline_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"offline", "true"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Screenshot_Element(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate and create a test element
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<div id="test" style="width:100px;height:50px;background:red;">Test</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	tmpFile := t.TempDir() + "/element.png"
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"screenshot", "--output", tmpFile, "--selector", "#test"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	// Verify file was created
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read screenshot: %v", err)
	}

	// Should be a valid PNG
	if len(data) < 8 || data[0] != 0x89 || data[1] != 0x50 {
		t.Error("output is not a valid PNG")
	}

	// Verify JSON output includes element info
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["selector"] != "#test" {
		t.Errorf("expected selector '#test', got %v", result["selector"])
	}
}

func TestRun_Styles_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate and create a styled element
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<div id="styled" style="color:red;font-size:16px;margin:10px;">Styled</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"styles", "#styled"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	styles, ok := result["styles"].(map[string]interface{})
	if !ok {
		t.Fatal("expected styles object in result")
	}

	// Check some computed style values
	if styles["color"] == nil {
		t.Error("expected color in styles")
	}
	if styles["fontSize"] == nil {
		t.Error("expected fontSize in styles")
	}
}

func TestRun_Styles_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"styles"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Layout_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate and create nested elements
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<div id="parent" style="padding:10px;"><span class="child" style="margin:5px;">A</span><span class="child">B</span></div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"layout", "#parent"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Should have bounds
	if result["bounds"] == nil {
		t.Error("expected bounds in result")
	}

	// Should have children
	children, ok := result["children"].([]interface{})
	if !ok {
		t.Fatal("expected children array in result")
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}

func TestRun_Layout_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"layout"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}
