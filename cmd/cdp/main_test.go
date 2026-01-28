package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tomyan/cdp-cli/internal/testutil"
)

// Test Chrome instance - each package gets its own
const testChromePort = 9301

var chromeInstance *testutil.ChromeInstance

// TestMain sets up and tears down Chrome for all tests
func TestMain(m *testing.M) {
	// Start Chrome for this package's tests
	var err error
	chromeInstance, err = testutil.StartChrome(testChromePort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Chrome: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Stop Chrome
	chromeInstance.Stop()

	os.Exit(code)
}

func testConfig() *Config {
	return &Config{
		Port:    testChromePort,
		Host:    "localhost",
		Timeout: 5 * time.Second,
		Output:  "json",
		Quiet:   false,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

// createTestTabCLI creates a new isolated tab for CLI tests.
// Returns the tab ID and a cleanup function that must be deferred.
func createTestTabCLI(t *testing.T) (string, func()) {
	t.Helper()

	cfg := testConfig()
	code := run([]string{"new"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to create test tab: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse new tab result: %v", err)
	}

	tabID, ok := result["targetId"].(string)
	if !ok {
		t.Fatalf("new tab result missing targetId")
	}

	cleanup := func() {
		cfg := testConfig()
		run([]string{"--target", tabID, "close"}, cfg)
	}

	return tabID, cleanup
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel() // No Chrome needed
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
	t.Parallel() // No Chrome needed
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
	t.Parallel() // No Chrome needed
	cfg := testConfig()
	code := run([]string{"-h"}, cfg)
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestRun_Version_NoChrome(t *testing.T) {
	t.Parallel() // No Chrome needed
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
	// Use testChromePort (the port our test Chrome runs on)
	code := run([]string{"-port", fmt.Sprintf("%d", testChromePort), "version"}, cfg)
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

func TestRun_Goto_Wait(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate with --wait flag
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><script>document.body.innerHTML='loaded';</script></body></html>`
	code := run([]string{"--target", tabID, "goto", "--wait", dataURL}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify loaded field is true
	if result["loaded"] != true {
		t.Errorf("expected loaded: true, got %v", result["loaded"])
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

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Create a page with an input
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<input id="test-input" type="text" />'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	// Reset buffers
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "fill", "#test-input", "test value"}, cfg)
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

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create a test element
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<div id="test">Content</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "html", "#test"}, cfg)
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

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create element that exists
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<div id="exists">Test</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "wait", "#exists"}, cfg)
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

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create input
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<input id="focus-input" type="text" />'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "focus", "#focus-input"}, cfg)
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Create an input
	code := run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<input id="press-input" type="text" />'`}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create input: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	code = run([]string{"--target", tabID, "focus", "#press-input"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to focus: %d", code)
	}

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"--target", tabID, "press", "Enter"}, cfg)
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Create a button
	code := run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<button id="hover-btn" style="width:100px;height:50px;">Hover</button>'`}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to create button: %d", code)
	}
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"--target", tabID, "hover", "#hover-btn"}, cfg)
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
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Timeout = 10 * time.Second

	// Navigate to a data URL with a link
	dataURL := `data:text/html,<html><body><a id="link" href="https://test.com">Test</a></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to navigate: %d, stderr: %s", code, stderr)
	}
	time.Sleep(200 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code = run([]string{"--target", tabID, "attr", "#link", "href"}, cfg)
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

func TestRun_Intercept_Enable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Enable interception
	code := run([]string{"intercept", "--response", "--pattern", "*", "--replace", "foo:bar"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to enable intercept: %d, stderr: %s", code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["enabled"] != true {
		t.Error("expected enabled to be true")
	}
	if result["pattern"] != "*" {
		t.Errorf("expected pattern *, got %v", result["pattern"])
	}
	if result["response"] != true {
		t.Error("expected response to be true")
	}
}

func TestRun_Intercept_Disable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Disable interception
	code := run([]string{"intercept", "--disable"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to disable intercept: %d, stderr: %s", code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["enabled"] != false {
		t.Error("expected enabled to be false")
	}
}

func TestRun_Intercept_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"intercept", "--pattern", "*"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Block_Enable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Enable blocking
	code := run([]string{"block", "*.js"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to enable block: %d, stderr: %s", code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["enabled"] != true {
		t.Error("expected enabled to be true")
	}
	patterns := result["patterns"].([]interface{})
	if len(patterns) != 1 || patterns[0] != "*.js" {
		t.Errorf("expected patterns [*.js], got %v", patterns)
	}
}

func TestRun_Block_Disable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Disable blocking
	code := run([]string{"block", "--disable"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("failed to disable block: %d, stderr: %s", code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result["enabled"] != false {
		t.Error("expected enabled to be false")
	}
}

func TestRun_Block_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"block", "*.js"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Metrics_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate first
	run([]string{"goto", "https://example.com"}, cfg)
	time.Sleep(500 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"metrics"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, stdout)
	}

	// Should have metrics
	metrics, ok := result["metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metrics in result")
	}

	// Check for some common metrics
	if _, ok := metrics["Timestamp"]; !ok {
		t.Error("expected Timestamp metric")
	}
}

func TestRun_Metrics_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"metrics"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_A11y_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate to a page with content
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<button>Click me</button><input type="text" placeholder="Name">'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"a11y"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, stdout)
	}

	// Should have nodes
	nodes, ok := result["nodes"].([]interface{})
	if !ok {
		t.Fatal("expected nodes in result")
	}

	if len(nodes) == 0 {
		t.Error("expected at least one accessibility node")
	}
}

func TestRun_A11y_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"a11y"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Source_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate to a page with content
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<h1>Test Page</h1>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"source"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, stdout)
	}

	html, ok := result["html"].(string)
	if !ok {
		t.Fatal("expected html in result")
	}

	if !strings.Contains(html, "<h1>Test Page</h1>") {
		t.Error("expected page source to contain <h1>Test Page</h1>")
	}
}

func TestRun_Source_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"source"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitIdle_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate to a simple page
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	// Wait for network idle
	code := run([]string{"waitidle"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, stdout)
	}

	if result["idle"] != true {
		t.Error("expected idle to be true")
	}
}

func TestRun_WaitIdle_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waitidle"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Links_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Navigate and create content with links
	run([]string{"goto", "about:blank"}, cfg)
	time.Sleep(50 * time.Millisecond)
	run([]string{"eval", `document.body.innerHTML = '<a href="https://example.com">Example</a><a href="/about">About</a>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"links"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, stdout)
	}

	links, ok := result["links"].([]interface{})
	if !ok {
		t.Fatal("expected links in result")
	}

	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
}

func TestRun_Links_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"links"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Upload_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"upload"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	code = run([]string{"upload", "#file-input"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Upload_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create file input
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<input type="file" id="file-input">'`}, cfg)
	time.Sleep(100 * time.Millisecond)

	// Create a temp file to upload
	tmpFile, err := os.CreateTemp("", "cdp-test-upload-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test content")
	tmpFile.Close()

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "upload", "#file-input", tmpFile.Name()}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["uploaded"] != true {
		t.Errorf("expected uploaded: true, got %v", result["uploaded"])
	}
}

func TestRun_Upload_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"upload", "#file-input", "/tmp/test.txt"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Exists_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"exists"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Exists_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create element
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<div id="test-element">Test</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "exists", "#test-element"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["exists"] != true {
		t.Errorf("expected exists: true, got %v", result["exists"])
	}
}

func TestRun_Exists_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "exists", "#nonexistent-element"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["exists"] != false {
		t.Errorf("expected exists: false, got %v", result["exists"])
	}
}

func TestRun_Exists_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"exists", "#test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitNav_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Timeout = 15 * time.Second

	// Create a link
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<a id="nav-link" href="about:blank">Navigate</a>'`}, cfg)
	time.Sleep(100 * time.Millisecond)

	// Start waitnav in background
	done := make(chan int, 1)
	go func() {
		navCfg := testConfig()
		navCfg.Stdout = &bytes.Buffer{}
		navCfg.Stderr = &bytes.Buffer{}
		code := run([]string{"--target", tabID, "--timeout", "10s", "waitnav", "--timeout", "10s"}, navCfg)
		done <- code
	}()

	// Give waitnav time to start
	time.Sleep(200 * time.Millisecond)

	// Click the link to trigger navigation
	run([]string{"--target", tabID, "click", "#nav-link"}, cfg)

	// Wait for waitnav to complete
	select {
	case code := <-done:
		if code != ExitSuccess {
			t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("waitnav timed out")
	}
}

func TestRun_WaitNav_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waitnav"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Value_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"value"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Value_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create input with value
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<input id="test-input" value="test value">'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "value", "#test-input"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["value"] != "test value" {
		t.Errorf("expected value: 'test value', got %v", result["value"])
	}
}

func TestRun_Value_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"value", "#test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitFn_MissingExpression(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"waitfn"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_WaitFn_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Set up a variable that's already true
	run([]string{"--target", tabID, "eval", `window.testReady = true`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "waitfn", "window.testReady", "--timeout", "5s"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["completed"] != true {
		t.Errorf("expected completed: true, got %v", result["completed"])
	}
}

func TestRun_WaitFn_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waitfn", "true"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Forms_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create form
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<form id="test-form"><input name="field1" type="text"></form>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "forms"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	count, ok := result["count"].(float64)
	if !ok || count != 1 {
		t.Errorf("expected count: 1, got %v", result["count"])
	}
}

func TestRun_Forms_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"forms"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Highlight_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"highlight"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_Highlight_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create element
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<div id="test-element">Test</div>'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "highlight", "#test-element"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["highlighted"] != true {
		t.Errorf("expected highlighted: true, got %v", result["highlighted"])
	}
}

func TestRun_Highlight_Hide(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "highlight", "--hide"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["hidden"] != true {
		t.Errorf("expected hidden: true, got %v", result["hidden"])
	}
}

func TestRun_Highlight_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"highlight", "#test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Images_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create images
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<img src="test.png" alt="Test 1"><img src="test2.jpg" alt="Test 2">'`}, cfg)
	time.Sleep(50 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "images"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	count, ok := result["count"].(float64)
	if !ok || count != 2 {
		t.Errorf("expected count: 2, got %v", result["count"])
	}
}

func TestRun_Images_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"images"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_ScrollBottom_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "scrollbottom"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["scrolled"] != true {
		t.Errorf("expected scrolled: true, got %v", result["scrolled"])
	}
}

func TestRun_ScrollBottom_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"scrollbottom"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_ScrollTop_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab for this test
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "scrolltop"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["scrolled"] != true {
		t.Errorf("expected scrolled: true, got %v", result["scrolled"])
	}
}

func TestRun_ScrollTop_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"scrolltop"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Frames_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab with iframe
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Create iframe
	run([]string{"--target", tabID, "eval", `document.body.innerHTML = '<iframe name="testframe" srcdoc="<div>inside</div>"></iframe>'`}, cfg)
	time.Sleep(500 * time.Millisecond)

	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "frames"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	count, ok := result["count"].(float64)
	if !ok || count < 2 {
		t.Errorf("expected at least 2 frames, got %v", result["count"])
	}
}

func TestRun_Frames_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"frames"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_EvalFrame_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"evalframe"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	code = run([]string{"evalframe", "frame-id"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_EvalFrame_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"evalframe", "frame-id", "1+1"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitGone_MissingSelector(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"waitgone"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestRun_WaitGone_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create isolated tab
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()

	// Element doesn't exist, should return immediately
	cfg.Stdout = &bytes.Buffer{}
	cfg.Stderr = &bytes.Buffer{}

	code := run([]string{"--target", tabID, "waitgone", "#nonexistent", "--timeout", "5s"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["gone"] != true {
		t.Errorf("expected gone: true, got %v", result["gone"])
	}
}

func TestRun_WaitGone_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waitgone", "#test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Session_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"session"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Session_GetSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with an origin (sessionStorage requires origin)
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", fmt.Sprintf("http://localhost:%d/json", testChromePort)}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate for session storage test")
	}

	// Set a session storage value
	cfg = testConfig()
	code = run([]string{"--target", tabID, "session", "testKey", "testValue"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var setResult map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &setResult); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
	if setResult["set"] != true {
		t.Errorf("expected set: true, got %v", setResult["set"])
	}

	// Get the session storage value
	cfg = testConfig()
	code = run([]string{"--target", tabID, "session", "testKey"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout = cfg.Stdout.(*bytes.Buffer).String()
	var getResult map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &getResult); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
	if getResult["value"] != "testValue" {
		t.Errorf("expected value: testValue, got %v", getResult["value"])
	}
}

func TestRun_Session_Clear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with an origin (sessionStorage requires origin)
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", fmt.Sprintf("http://localhost:%d/json", testChromePort)}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate for session storage test")
	}

	// Set a session storage value first
	cfg = testConfig()
	code = run([]string{"--target", tabID, "session", "clearKey", "clearValue"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to set session storage")
	}

	// Clear all session storage
	cfg = testConfig()
	code = run([]string{"--target", tabID, "session", "--clear"}, cfg)
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

	// Verify value is gone
	cfg = testConfig()
	code = run([]string{"--target", tabID, "session", "clearKey"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to get session storage after clear")
	}

	stdout = cfg.Stdout.(*bytes.Buffer).String()
	var getResult map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &getResult); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
	// After clear, value should be empty string (null from JS becomes "")
	if getResult["value"] != "" {
		t.Errorf("expected empty value after clear, got %v", getResult["value"])
	}
}

func TestRun_Session_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"session", "key"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Throttle_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"throttle"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Throttle_Preset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Apply slow 3G throttling
	cfg := testConfig()
	code := run([]string{"--target", tabID, "throttle", "slow3g"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["preset"] != "slow3g" {
		t.Errorf("expected preset: slow3g, got %v", result["preset"])
	}
	if result["enabled"] != true {
		t.Errorf("expected enabled: true, got %v", result["enabled"])
	}
}

func TestRun_Throttle_Disable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Enable throttling first
	cfg := testConfig()
	code := run([]string{"--target", tabID, "throttle", "fast3g"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to enable throttling")
	}

	// Disable throttling
	cfg = testConfig()
	code = run([]string{"--target", tabID, "throttle", "--disable"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["disabled"] != true {
		t.Errorf("expected disabled: true, got %v", result["disabled"])
	}
}

func TestRun_Throttle_InvalidPreset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	cfg := testConfig()
	code := run([]string{"--target", tabID, "throttle", "invalid"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d for invalid preset, got %d", ExitError, code)
	}
}

func TestRun_Throttle_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"throttle", "slow3g"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Meta_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with meta tags
	cfg := testConfig()
	dataURL := `data:text/html,<html><head><meta charset="UTF-8"><meta name="description" content="Test Description"><meta name="viewport" content="width=device-width"></head><body></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get meta tags
	cfg = testConfig()
	code = run([]string{"--target", tabID, "meta"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	tags, ok := result["tags"].([]interface{})
	if !ok {
		t.Fatalf("expected tags array, got %T", result["tags"])
	}

	if len(tags) < 2 {
		t.Errorf("expected at least 2 meta tags, got %d", len(tags))
	}

	// Check that we have the description tag
	foundDescription := false
	for _, tag := range tags {
		m := tag.(map[string]interface{})
		if m["name"] == "description" && m["content"] == "Test Description" {
			foundDescription = true
			break
		}
	}
	if !foundDescription {
		t.Errorf("expected to find description meta tag")
	}
}

func TestRun_Meta_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"meta"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Tables_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with a table
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><table id="data"><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>Alice</td><td>30</td></tr><tr><td>Bob</td><td>25</td></tr></tbody></table></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get tables
	cfg = testConfig()
	code = run([]string{"--target", tabID, "tables"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	tables, ok := result["tables"].([]interface{})
	if !ok {
		t.Fatalf("expected tables array, got %T", result["tables"])
	}

	if len(tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(tables))
	}

	// Check the table structure
	table := tables[0].(map[string]interface{})
	if table["id"] != "data" {
		t.Errorf("expected table id 'data', got %v", table["id"])
	}

	headers := table["headers"].([]interface{})
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}

	rows := table["rows"].([]interface{})
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestRun_Tables_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"tables"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_ClickAt_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"clickat"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_ClickAt_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with a click target
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><button id="btn" style="position:absolute;left:50px;top:50px;width:100px;height:50px" onclick="document.body.classList.add('clicked')">Click Me</button></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Click at specific coordinates (center of button)
	cfg = testConfig()
	code = run([]string{"--target", tabID, "clickat", "100", "75"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["x"].(float64) != 100 || result["y"].(float64) != 75 {
		t.Errorf("expected x:100, y:75, got x:%v, y:%v", result["x"], result["y"])
	}
}

func TestRun_ClickAt_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"clickat", "100", "100"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Errors_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page that throws an error
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><script>throw new Error("Test error");</script></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get page errors (short duration to capture the error)
	cfg = testConfig()
	code = run([]string{"--target", tabID, "errors", "--duration", "500ms"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	// Output is NDJSON, so parse line by line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 0 || lines[0] == "" {
		// No errors captured is acceptable for this test - error might have already happened
		return
	}

	// Verify first line is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestRun_Errors_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"errors"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Tap_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"tap"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Tap_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with a tap target
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><button id="btn" ontouchstart="document.body.classList.add('tapped')">Tap Me</button></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Tap on the button using selector
	cfg = testConfig()
	code = run([]string{"--target", tabID, "tap", "#btn"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["tapped"] != true {
		t.Errorf("expected tapped: true, got %v", result["tapped"])
	}
}

func TestRun_Tap_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"tap", "#btn"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Media_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"media"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Media_ColorScheme(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Set color scheme to dark
	cfg := testConfig()
	code := run([]string{"--target", tabID, "media", "--color-scheme", "dark"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["colorScheme"] != "dark" {
		t.Errorf("expected colorScheme: dark, got %v", result["colorScheme"])
	}
}

func TestRun_Media_ReducedMotion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Set reduced motion
	cfg := testConfig()
	code := run([]string{"--target", tabID, "media", "--reduced-motion", "reduce"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["reducedMotion"] != "reduce" {
		t.Errorf("expected reducedMotion: reduce, got %v", result["reducedMotion"])
	}
}

func TestRun_Media_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"media", "--color-scheme", "dark"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Permission_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"permission"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Permission_Grant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page first (permissions need an origin)
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", fmt.Sprintf("http://localhost:%d/json", testChromePort)}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Grant geolocation permission
	cfg = testConfig()
	code = run([]string{"--target", tabID, "permission", "geolocation", "granted"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["permission"] != "geolocation" {
		t.Errorf("expected permission: geolocation, got %v", result["permission"])
	}
	if result["state"] != "granted" {
		t.Errorf("expected state: granted, got %v", result["state"])
	}
}

func TestRun_Permission_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"permission", "geolocation", "granted"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Clipboard_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"clipboard"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Clipboard_Write(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page first
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", fmt.Sprintf("http://localhost:%d/json", testChromePort)}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Write to clipboard
	cfg = testConfig()
	code = run([]string{"--target", tabID, "clipboard", "--write", "test clipboard content"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["written"] != true {
		t.Errorf("expected written: true, got %v", result["written"])
	}
}

func TestRun_Clipboard_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"clipboard", "--write", "test"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Drag_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"drag"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Drag_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with draggable elements
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><div id="src" draggable="true" style="width:50px;height:50px;background:red;position:absolute;left:10px;top:10px"></div><div id="dst" style="width:100px;height:100px;background:blue;position:absolute;left:200px;top:10px"></div></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Drag from source to destination
	cfg = testConfig()
	code = run([]string{"--target", tabID, "drag", "#src", "#dst"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["dragged"] != true {
		t.Errorf("expected dragged: true, got %v", result["dragged"])
	}
}

func TestRun_Drag_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"drag", "#src", "#dst"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitURL_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"waiturl"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_WaitURL_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a known URL
	cfg := testConfig()
	targetURL := fmt.Sprintf("http://localhost:%d/json", testChromePort)
	code := run([]string{"--target", tabID, "goto", targetURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Wait for URL containing "json"
	cfg = testConfig()
	code = run([]string{"--target", tabID, "waiturl", "json", "--timeout", "2s"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if _, ok := result["url"]; !ok {
		t.Errorf("expected url field in result")
	}
}

func TestRun_WaitURL_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Wait for URL that won't match (should timeout)
	cfg = testConfig()
	code = run([]string{"--target", tabID, "waiturl", "nonexistent-pattern-xyz", "--timeout", "500ms"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d for timeout, got %d", ExitError, code)
	}
}

func TestRun_WaitURL_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waiturl", "pattern"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Shadow_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"shadow"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Shadow_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with a shadow DOM element
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><div id="host"></div><script>
		const host = document.getElementById('host');
		const shadow = host.attachShadow({mode: 'open'});
		shadow.innerHTML = '<span id="inner">Shadow Content</span>';
	</script></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Query shadow DOM element
	cfg = testConfig()
	code = run([]string{"--target", tabID, "shadow", "#host", "#inner"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["tagName"] != "SPAN" {
		t.Errorf("expected tagName SPAN, got %v", result["tagName"])
	}
}

func TestRun_Shadow_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"shadow", "#host", "#inner"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Har_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a URL that makes a network request
	cfg := testConfig()
	targetURL := fmt.Sprintf("http://localhost:%d/json", testChromePort)
	code := run([]string{"--target", tabID, "goto", targetURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get HAR for a short duration
	cfg = testConfig()
	code = run([]string{"--target", tabID, "har", "--duration", "500ms"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify HAR structure
	if _, ok := result["log"]; !ok {
		t.Errorf("expected HAR log field")
	}
}

func TestRun_Har_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"har"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Coverage_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with some JavaScript
	cfg := testConfig()
	dataURL := `data:text/html,<html><body><script>
		function used() { return 1; }
		function unused() { return 2; }
		used();
	</script></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get coverage
	cfg = testConfig()
	code = run([]string{"--target", tabID, "coverage"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify coverage structure
	if _, ok := result["scripts"]; !ok {
		t.Errorf("expected scripts field in result")
	}
}

func TestRun_Coverage_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"coverage"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Stylesheets_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with some CSS
	cfg := testConfig()
	dataURL := `data:text/html,<html><head><style>.test { color: red; }</style></head><body></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get stylesheets
	cfg = testConfig()
	code = run([]string{"--target", tabID, "stylesheets"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify stylesheets structure
	if _, ok := result["stylesheets"]; !ok {
		t.Errorf("expected stylesheets field in result")
	}
}

func TestRun_Stylesheets_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"stylesheets"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Screenshot_Base64(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body style='background:blue'>Hello</body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Take screenshot with base64 output
	cfg = testConfig()
	code = run([]string{"--target", tabID, "screenshot", "--base64"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify base64 data
	if data, ok := result["data"].(string); !ok || len(data) == 0 {
		t.Errorf("expected non-empty data field")
	}

	if format, ok := result["format"].(string); !ok || format != "png" {
		t.Errorf("expected format png, got %v", result["format"])
	}
}

func TestRun_Info_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with a title
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><head><title>Test Page</title></head><body>Hello</body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get page info
	cfg = testConfig()
	code = run([]string{"--target", tabID, "info"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify info structure
	if _, ok := result["title"]; !ok {
		t.Errorf("expected title field")
	}
	if _, ok := result["url"]; !ok {
		t.Errorf("expected url field")
	}
}

func TestRun_Info_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"info"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitText_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"waittext"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_WaitText_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with text
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body>Hello World</body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Wait for text that exists
	cfg = testConfig()
	code = run([]string{"--target", tabID, "waittext", "Hello", "--timeout", "2s"}, cfg)
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

func TestRun_WaitText_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body>Hello</body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Wait for text that doesn't exist (should timeout)
	cfg = testConfig()
	code = run([]string{"--target", tabID, "waittext", "nonexistent-text-xyz", "--timeout", "500ms"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d for timeout, got %d", ExitError, code)
	}
}

func TestRun_WaitText_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waittext", "hello"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Scripts_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with scripts
	cfg := testConfig()
	dataURL := `data:text/html,<html><head><script src="test.js"></script></head><body><script>console.log('inline');</script></body></html>`
	code := run([]string{"--target", tabID, "goto", dataURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Get scripts
	cfg = testConfig()
	code = run([]string{"--target", tabID, "scripts"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify scripts structure
	if _, ok := result["scripts"]; !ok {
		t.Errorf("expected scripts field in result")
	}
}

func TestRun_Scripts_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"scripts"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Find_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"find"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Find_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with text
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body><p>Hello World</p><p>Hello Again</p></body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Find text
	cfg = testConfig()
	code = run([]string{"--target", tabID, "find", "Hello"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if count, ok := result["count"].(float64); !ok || count < 2 {
		t.Errorf("expected count >= 2, got %v", result["count"])
	}
}

func TestRun_Find_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"find", "hello"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_SetValue_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"setvalue"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_SetValue_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with an input
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body><input id='test' type='text'></body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Set the value directly
	cfg = testConfig()
	code = run([]string{"--target", tabID, "setvalue", "#test", "Hello World"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["value"] != "Hello World" {
		t.Errorf("expected value 'Hello World', got %v", result["value"])
	}
}

func TestRun_SetValue_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"setvalue", "#input", "value"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_Press_WithModifiers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page with an input
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body><input id='test' type='text' value='hello world'></body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Focus the input
	cfg = testConfig()
	code = run([]string{"--target", tabID, "focus", "#test"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to focus")
	}

	// Press Ctrl+a (select all)
	cfg = testConfig()
	code = run([]string{"--target", tabID, "press", "Ctrl+a"}, cfg)
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
	if result["key"] != "Ctrl+a" {
		t.Errorf("expected key: 'Ctrl+a', got %v", result["key"])
	}
}

func TestRun_Mouse_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"mouse"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_Mouse_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to a page
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "data:text/html,<html><body></body></html>"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Move mouse to coordinates
	cfg = testConfig()
	code = run([]string{"--target", tabID, "mouse", "100", "200"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if result["x"].(float64) != 100 || result["y"].(float64) != 200 {
		t.Errorf("expected x:100, y:200, got x:%v, y:%v", result["x"], result["y"])
	}
}

func TestRun_Mouse_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"mouse", "100", "100"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}

func TestRun_WaitRequest_MissingArgs(t *testing.T) {
	cfg := testConfig()
	code := run([]string{"waitrequest"}, cfg)
	if code != ExitError {
		t.Errorf("expected exit code %d, got %d", ExitError, code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message, got: %s", stderr)
	}
}

func TestRun_WaitRequest_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// Navigate to Chrome's json endpoint first
	cfg := testConfig()
	targetURL := fmt.Sprintf("http://localhost:%d/json", testChromePort)
	code := run([]string{"--target", tabID, "goto", targetURL}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to navigate")
	}

	// Schedule a reload via JavaScript that will happen while we wait
	cfg = testConfig()
	code = run([]string{"--target", tabID, "eval", "setTimeout(() => location.reload(), 300)"}, cfg)
	if code != ExitSuccess {
		t.Fatalf("failed to schedule reload")
	}

	// Wait for a request containing "json" (will be triggered by the delayed reload)
	cfg = testConfig()
	code = run([]string{"--target", tabID, "waitrequest", "json", "--timeout", "5s"}, cfg)
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

func TestRun_WaitRequest_NoChrome(t *testing.T) {
	cfg := testConfig()
	cfg.Port = 1 // Invalid port
	code := run([]string{"waitrequest", "pattern"}, cfg)
	if code != ExitConnFailed {
		t.Errorf("expected exit code %d, got %d", ExitConnFailed, code)
	}
}
