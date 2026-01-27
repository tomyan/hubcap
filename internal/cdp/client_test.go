package cdp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tomyan/cdp-cli/internal/cdp"
	"github.com/tomyan/cdp-cli/internal/testutil"
)

// Test Chrome instance - each package gets its own
const testChromePort = 9300

var (
	chromeInstance *testutil.ChromeInstance
	sharedClient   *cdp.Client
	sharedClientMu sync.Mutex
	clientInitErr  error
)

// TestMain sets up and tears down shared resources for all tests
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

	// Clean up shared client if it was created
	sharedClientMu.Lock()
	if sharedClient != nil {
		sharedClient.Close()
	}
	sharedClientMu.Unlock()

	// Stop Chrome
	chromeInstance.Stop()

	os.Exit(code)
}

// getSharedClient returns a shared client for tests that don't modify browser state.
// The client is lazily initialized on first use and reused across tests.
// Tests that modify state should create their own client.
func getSharedClient(t *testing.T) *cdp.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	sharedClientMu.Lock()
	defer sharedClientMu.Unlock()

	if clientInitErr != nil {
		t.Fatalf("shared client initialization failed previously: %v", clientInitErr)
	}

	if sharedClient == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var err error
		sharedClient, err = cdp.Connect(ctx, "localhost", testChromePort)
		if err != nil {
			clientInitErr = err
			t.Fatalf("failed to connect shared client: %v", err)
		}
	}

	return sharedClient
}

// createTestTab creates a new isolated tab for tests that modify state.
// Returns the tab ID and a cleanup function that must be deferred.
// This prevents tests from interfering with each other's page state.
func createTestTab(t *testing.T, client *cdp.Client, ctx context.Context) (string, func()) {
	t.Helper()

	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create test tab: %v", err)
	}

	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		client.CloseTab(cleanupCtx, tabID)
	}

	return tabID, cleanup
}

func TestClient_Version_ReturnsVersionInfo(t *testing.T) {
	t.Parallel() // Read-only test, safe to run in parallel
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := client.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	// Should have browser field
	if version.Browser == "" {
		t.Error("expected non-empty Browser field")
	}

	// Should have protocol version
	if version.ProtocolVersion == "" {
		t.Error("expected non-empty ProtocolVersion field")
	}
}

func TestClient_Version_JSONSerializable(t *testing.T) {
	t.Parallel() // Read-only test, safe to run in parallel
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := client.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	// Should be JSON serializable
	data, err := json.Marshal(version)
	if err != nil {
		t.Fatalf("failed to marshal version: %v", err)
	}

	// Should contain expected fields
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := m["browser"]; !ok {
		t.Error("JSON should contain 'browser' field")
	}
	if _, ok := m["protocol"]; !ok {
		t.Error("JSON should contain 'protocol' field")
	}
}

func TestClient_Connect_FailsWithBadPort(t *testing.T) {
	t.Parallel() // No Chrome connection, safe to run in parallel
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Port 1 should fail to connect
	_, err := cdp.Connect(ctx, "localhost", 1)
	if err == nil {
		t.Error("expected connection to fail on port 1")
	}
}

func TestClient_Connect_FailsWithBadHost(t *testing.T) {
	t.Parallel() // No Chrome connection, safe to run in parallel
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := cdp.Connect(ctx, "nonexistent.invalid", testChromePort)
	if err == nil {
		t.Error("expected connection to fail with invalid host")
	}
}

func TestClient_WebSocketURL(t *testing.T) {
	t.Parallel() // Read-only test
	client := getSharedClient(t)

	wsURL := client.WebSocketURL()
	if wsURL == "" {
		t.Error("expected non-empty WebSocket URL")
	}
	if !strings.HasPrefix(wsURL, "ws://") {
		t.Errorf("expected WebSocket URL to start with ws://, got %s", wsURL)
	}
}

func TestClient_Call_ReturnsErrorOnClosed(t *testing.T) {
	// Uses own client - not parallel due to Chrome resource contention
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Close connection
	client.Close()

	// Try to call after close
	_, err = client.Call(ctx, "Browser.getVersion", nil)
	if !errors.Is(err, cdp.ErrConnectionClosed) {
		t.Errorf("expected ErrConnectionClosed, got %v", err)
	}
}

func TestClient_Call_InvalidMethod(t *testing.T) {
	t.Parallel() // Read-only test
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call invalid method
	_, err := client.Call(ctx, "Invalid.nonExistentMethod", nil)
	if err == nil {
		t.Error("expected error for invalid method")
	}

	// Should be a CDP error
	if !errors.Is(err, cdp.ErrCDPError) {
		t.Errorf("expected ErrCDPError, got %v", err)
	}
}

func TestClient_Targets_ReturnsPages(t *testing.T) {
	t.Parallel() // Read-only test
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targets, err := client.Targets(ctx)
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}

	// Should return a slice (may be empty)
	if targets == nil {
		t.Error("expected non-nil targets slice")
	}
}

func TestClient_Targets_JSONSerializable(t *testing.T) {
	t.Parallel() // Read-only test
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targets, err := client.Targets(ctx)
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}

	data, err := json.Marshal(targets)
	if err != nil {
		t.Fatalf("failed to marshal targets: %v", err)
	}

	// Should be a JSON array
	if len(data) == 0 || data[0] != '[' {
		t.Errorf("expected JSON array, got: %s", string(data))
	}
}

func TestTargetInfo_Fields(t *testing.T) {
	t.Parallel() // Read-only test
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targets, err := client.Targets(ctx)
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}

	if len(targets) == 0 {
		t.Skip("no targets available")
	}

	// Check first target has expected fields
	target := targets[0]
	if target.ID == "" {
		t.Error("expected non-empty ID")
	}
	if target.Type == "" {
		t.Error("expected non-empty Type")
	}
}

func TestClient_Pages_ReturnsOnlyPages(t *testing.T) {
	t.Parallel() // Read-only test
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}

	// All returned targets should be pages
	for _, p := range pages {
		if p.Type != "page" {
			t.Errorf("expected type 'page', got %q", p.Type)
		}
	}
}

func TestClient_Navigate_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	result, err := client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	if result.URL == "" {
		t.Error("expected non-empty URL")
	}
	if !strings.Contains(result.URL, "example.com") {
		t.Errorf("expected URL to contain example.com, got %s", result.URL)
	}
}

func TestClient_Navigate_InvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Navigate to invalid URL - should still work but result in error page
	result, err := client.Navigate(ctx, tabID, "not-a-valid-url")
	// This may or may not error depending on Chrome version
	if err == nil && result.URL == "" {
		t.Error("expected either error or non-empty URL")
	}
}

func TestNavigateResult_JSONSerializable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	result, err := client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := m["url"]; !ok {
		t.Error("expected 'url' field in JSON")
	}
}

func TestClient_Screenshot_ReturnsPNG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with content
	dataURL := `data:text/html,<html><body><h1>Screenshot Test</h1></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	data, err := client.Screenshot(ctx, tabID, cdp.ScreenshotOptions{
		Format: "png",
	})
	if err != nil {
		t.Fatalf("failed to take screenshot: %v", err)
	}

	// PNG magic bytes
	if len(data) < 8 {
		t.Fatal("screenshot data too small")
	}
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i, b := range pngMagic {
		if data[i] != b {
			t.Fatalf("not a valid PNG: byte %d is %x, expected %x", i, data[i], b)
		}
	}
}

func TestClient_Screenshot_ReturnsJPEG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with content
	dataURL := `data:text/html,<html><body><h1>Screenshot Test</h1></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	data, err := client.Screenshot(ctx, tabID, cdp.ScreenshotOptions{
		Format:  "jpeg",
		Quality: 80,
	})
	if err != nil {
		t.Fatalf("failed to take screenshot: %v", err)
	}

	// JPEG magic bytes
	if len(data) < 2 {
		t.Fatal("screenshot data too small")
	}
	if data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatalf("not a valid JPEG: got %x %x", data[0], data[1])
	}
}

func TestClient_Eval_SimpleExpression(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	result, err := client.Eval(ctx, tabID, "1 + 2")
	if err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Should return EvalResult with value 3
	if result.Value == nil {
		t.Error("expected non-nil value")
	}

	// Value should be number 3
	if v, ok := result.Value.(float64); !ok || v != 3 {
		t.Errorf("expected value 3, got %v (%T)", result.Value, result.Value)
	}
}

func TestClient_Eval_StringExpression(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	result, err := client.Eval(ctx, tabID, "'hello' + ' world'")
	if err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if v, ok := result.Value.(string); !ok || v != "hello world" {
		t.Errorf("expected 'hello world', got %v", result.Value)
	}
}

func TestClient_Eval_JSONSerializable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	result, err := client.Eval(ctx, tabID, "({a: 1, b: 'test'})")
	if err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if m["value"] == nil {
		t.Error("expected 'value' in JSON")
	}
}

func TestClient_Eval_JSException(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	_, err = client.Eval(ctx, tabID, "throw new Error('test error')")
	if err == nil {
		t.Error("expected error for thrown exception")
	}

	if !strings.Contains(err.Error(), "JS exception") {
		t.Errorf("expected 'JS exception' in error, got: %v", err)
	}
}

func TestClient_Query_FindsElement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	dataURL := `data:text/html,<html><body><div id="test">Test</div></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Query for body element
	result, err := client.Query(ctx, tabID, "body")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if result.NodeID == 0 {
		t.Error("expected non-zero nodeId")
	}
	if result.TagName != "BODY" {
		t.Errorf("expected tagName 'BODY', got %q", result.TagName)
	}
}

func TestClient_Query_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Query for non-existent element
	result, err := client.Query(ctx, tabID, "#nonexistent-element-12345")
	if err != nil {
		t.Fatalf("expected nil error for not found, got: %v", err)
	}

	if result.NodeID != 0 {
		t.Errorf("expected nodeId 0 for not found, got %d", result.NodeID)
	}
}

func TestClient_Query_JSONSerializable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	result, err := client.Query(ctx, tabID, "body")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if m["nodeId"] == nil {
		t.Error("expected 'nodeId' in JSON")
	}
	if m["tagName"] == nil {
		t.Error("expected 'tagName' in JSON")
	}
}

func TestClient_Click_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with a clickable link
	dataURL := `data:text/html,<html><body><a id="link" href="about:blank">Click me</a><script>window.clicked=false;document.getElementById('link').addEventListener('click',e=>{e.preventDefault();window.clicked=true});</script></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Click the link
	err = client.Click(ctx, tabID, "#link")
	if err != nil {
		t.Fatalf("failed to click: %v", err)
	}

	// Verify the click event was handled
	result, err := client.Eval(ctx, tabID, `window.clicked`)
	if err != nil {
		t.Fatalf("failed to verify click: %v", err)
	}
	if result.Value != true {
		t.Errorf("expected clicked: true, got %v", result.Value)
	}
}

func TestClient_Click_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Try to click non-existent element
	err = client.Click(ctx, tabID, "#nonexistent-element-12345")
	if err == nil {
		t.Error("expected error for non-existent element")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestClient_Fill_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with an input field
	dataURL := `data:text/html,<html><body><input id="test-input" type="text" /></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Fill the input
	err = client.Fill(ctx, tabID, "#test-input", "hello world")
	if err != nil {
		t.Fatalf("failed to fill: %v", err)
	}

	// Verify the value
	result, err := client.Eval(ctx, tabID, `document.querySelector('#test-input').value`)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	if result.Value != "hello world" {
		t.Errorf("expected 'hello world', got %v", result.Value)
	}
}

func TestClient_Fill_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Try to fill non-existent element
	err = client.Fill(ctx, tabID, "#nonexistent-input-12345", "test")
	if err == nil {
		t.Error("expected error for non-existent element")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestClient_GetHTML_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with test content
	dataURL := `data:text/html,<html><body><div id="test"><span>Hello</span></div></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Get HTML
	html, err := client.GetHTML(ctx, tabID, "#test")
	if err != nil {
		t.Fatalf("failed to get HTML: %v", err)
	}

	if !strings.Contains(html, "<span>Hello</span>") {
		t.Errorf("expected HTML to contain '<span>Hello</span>', got %q", html)
	}
	if !strings.Contains(html, `id="test"`) {
		t.Errorf("expected HTML to contain 'id=\"test\"', got %q", html)
	}
}

func TestClient_GetHTML_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	_, err = client.GetHTML(ctx, tabID, "#nonexistent-12345")
	if err == nil {
		t.Error("expected error for non-existent element")
	}
}

func TestClient_WaitFor_ImmediateMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with element already present
	dataURL := `data:text/html,<html><body><div id="exists">Test</div></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Wait should return immediately
	err = client.WaitFor(ctx, tabID, "#exists", 5*time.Second)
	if err != nil {
		t.Fatalf("WaitFor failed: %v", err)
	}
}

func TestClient_WaitFor_DelayedAppear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Set up delayed element creation (500ms)
	_, err = client.Eval(ctx, tabID, `
		document.body.innerHTML = '';
		setTimeout(() => {
			document.body.innerHTML = '<div id="delayed">Appeared</div>';
		}, 500);
	`)
	if err != nil {
		t.Fatalf("failed to set up delayed element: %v", err)
	}

	// Wait should poll and find it
	err = client.WaitFor(ctx, tabID, "#delayed", 5*time.Second)
	if err != nil {
		t.Fatalf("WaitFor failed: %v", err)
	}
}

func TestClient_WaitFor_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Wait for non-existent element with short timeout
	err = client.WaitFor(ctx, tabID, "#never-exists", 500*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected 'timeout' in error, got: %v", err)
	}
}

func TestClient_GetText_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with test content
	dataURL := `data:text/html,<html><body><div id="test">Hello <span>World</span>!</div></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	text, err := client.GetText(ctx, tabID, "#test")
	if err != nil {
		t.Fatalf("failed to get text: %v", err)
	}

	// Should get concatenated text content
	if text != "Hello World!" {
		t.Errorf("expected 'Hello World!', got %q", text)
	}
}

func TestClient_Type_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with an input and keydown counter
	dataURL := `data:text/html,<html><body><input id="test-input" type="text" /><script>window.keydownCount=0;document.getElementById('test-input').addEventListener('keydown',()=>{window.keydownCount++});</script></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Focus the input first
	_, err = client.Eval(ctx, tabID, `document.querySelector('#test-input').focus()`)
	if err != nil {
		t.Fatalf("failed to focus input: %v", err)
	}

	// Type "abc" character by character
	err = client.Type(ctx, tabID, "abc")
	if err != nil {
		t.Fatalf("failed to type: %v", err)
	}

	// Verify the value
	result, err := client.Eval(ctx, tabID, `document.querySelector('#test-input').value`)
	if err != nil {
		t.Fatalf("failed to verify value: %v", err)
	}

	if result.Value != "abc" {
		t.Errorf("expected 'abc', got %v", result.Value)
	}

	// Verify keydown events were fired (should be 3, one per character)
	countResult, err := client.Eval(ctx, tabID, `window.keydownCount`)
	if err != nil {
		t.Fatalf("failed to get keydown count: %v", err)
	}

	if count, ok := countResult.Value.(float64); !ok || count != 3 {
		t.Errorf("expected 3 keydown events, got %v", countResult.Value)
	}
}

func TestClient_CaptureConsole_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Start capturing console messages
	messages, stopCapture, err := client.CaptureConsole(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to start console capture: %v", err)
	}
	defer stopCapture() // Clean up resources when test ends

	// Trigger some console messages via eval
	_, err = client.Eval(ctx, tabID, `
		console.log("test log");
		console.warn("test warning");
		console.error("test error");
	`)
	if err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Give time for messages to arrive
	time.Sleep(100 * time.Millisecond)

	// Check that we received messages
	select {
	case msg := <-messages:
		if msg.Text == "" {
			t.Error("expected non-empty message text")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for console message")
	}
}

func TestClient_GetCookies_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab and navigate
	tabID, err := client.NewTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(500 * time.Millisecond)

	// Get cookies - should return a slice (may be empty)
	cookies, err := client.GetCookies(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}

	// Should return a non-nil slice
	if cookies == nil {
		t.Error("expected non-nil cookies slice")
	}
}

func TestClient_SetCookie_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab and navigate to example.com
	tabID, err := client.NewTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(1 * time.Second) // Wait for page to fully load

	// Set a cookie
	err = client.SetCookie(ctx, tabID, cdp.Cookie{
		Name:   "test_cookie",
		Value:  "test_value",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie: %v", err)
	}

	// Verify cookie was set
	cookies, err := client.GetCookies(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}

	found := false
	for _, c := range cookies {
		if c.Name == "test_cookie" && c.Value == "test_value" {
			found = true
			break
		}
	}

	if !found {
		t.Error("cookie was not set correctly")
	}
}

func TestClient_PrintToPDF_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with content
	dataURL := `data:text/html,<html><body><h1>Test PDF</h1><p>This is test content for PDF generation.</p></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Generate PDF
	data, err := client.PrintToPDF(ctx, tabID, cdp.PDFOptions{})
	if err != nil {
		t.Fatalf("failed to print to PDF: %v", err)
	}

	// PDF magic bytes: %PDF-
	if len(data) < 5 {
		t.Fatal("PDF data too small")
	}
	pdfMagic := []byte{0x25, 0x50, 0x44, 0x46, 0x2D} // %PDF-
	for i, b := range pdfMagic {
		if data[i] != b {
			t.Fatalf("not a valid PDF: byte %d is %x, expected %x", i, data[i], b)
		}
	}
}

func TestClient_DeleteCookie_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab and navigate to example.com
	tabID, err := client.NewTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(1 * time.Second) // Wait for page to fully load

	// Set a cookie first
	err = client.SetCookie(ctx, tabID, cdp.Cookie{
		Name:   "delete_test",
		Value:  "test_value",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie: %v", err)
	}

	// Verify cookie exists
	cookies, err := client.GetCookies(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}
	found := false
	for _, c := range cookies {
		if c.Name == "delete_test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("cookie was not set")
	}

	// Delete the cookie
	err = client.DeleteCookie(ctx, tabID, "delete_test", "example.com")
	if err != nil {
		t.Fatalf("failed to delete cookie: %v", err)
	}

	// Verify cookie is gone
	cookies, err = client.GetCookies(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get cookies after delete: %v", err)
	}
	for _, c := range cookies {
		if c.Name == "delete_test" {
			t.Error("cookie was not deleted")
		}
	}
}

func TestClient_ClearCookies_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab and navigate
	tabID, err := client.NewTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(500 * time.Millisecond)

	// Set some cookies
	err = client.SetCookie(ctx, tabID, cdp.Cookie{
		Name:   "clear_test1",
		Value:  "value1",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie 1: %v", err)
	}
	err = client.SetCookie(ctx, tabID, cdp.Cookie{
		Name:   "clear_test2",
		Value:  "value2",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie 2: %v", err)
	}

	// Clear all cookies
	err = client.ClearCookies(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to clear cookies: %v", err)
	}

	// Verify cookies are gone
	cookies, err := client.GetCookies(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}

	if len(cookies) > 0 {
		t.Errorf("expected no cookies after clear, got %d", len(cookies))
	}
}

func TestClient_Focus_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with an input element
	dataURL := `data:text/html,<html><body><input id="focus-test" type="text" /></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Focus the element
	err = client.Focus(ctx, tabID, "#focus-test")
	if err != nil {
		t.Fatalf("failed to focus: %v", err)
	}

	// Verify focus via JS
	result, err := client.Eval(ctx, tabID, `document.activeElement.id`)
	if err != nil {
		t.Fatalf("failed to verify focus: %v", err)
	}

	if result.Value != "focus-test" {
		t.Errorf("expected focused element id 'focus-test', got %v", result.Value)
	}
}

func TestClient_CaptureNetwork_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Start capturing network events
	events, stopCapture, err := client.CaptureNetwork(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to start network capture: %v", err)
	}
	defer stopCapture() // Clean up resources when test ends

	// Navigate to a page to trigger network requests
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Navigate(ctx, tabID, "https://example.com")
	}()

	// Wait for at least one request event
	timeout := time.After(5 * time.Second)
	gotRequest := false
	for !gotRequest {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			if event.Type == "request" && strings.Contains(event.URL, "example.com") {
				gotRequest = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for network events")
		}
	}
}

func TestClient_PressKey_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Use data: URL to create a self-contained test page in isolated tab
	dataURL := `data:text/html,<html><body><input id="test-input" type="text"/><script>window.lastKey='none';document.getElementById('test-input').addEventListener('keydown',e=>{window.lastKey=e.key});</script></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Focus the input
	err = client.Focus(ctx, tabID, "#test-input")
	if err != nil {
		t.Fatalf("failed to focus: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Press the Enter key
	err = client.PressKey(ctx, tabID, "Enter")
	if err != nil {
		t.Fatalf("failed to press key: %v", err)
	}
	time.Sleep(300 * time.Millisecond) // Wait for key event to be processed

	// Verify the key was pressed
	result, err := client.Eval(ctx, tabID, `window.lastKey`)
	if err != nil {
		t.Fatalf("failed to verify key: %v", err)
	}

	if result.Value != "Enter" {
		t.Errorf("expected lastKey 'Enter', got %v", result.Value)
	}
}

func TestClient_Hover_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with self-contained page
	dataURL := `data:text/html,<html><body><button id="hover-btn" style="width:100px;height:50px;">Hover me</button><script>window.hovered=false;document.getElementById('hover-btn').addEventListener('mouseenter',()=>{window.hovered=true});</script></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Hover over the button
	err = client.Hover(ctx, tabID, "#hover-btn")
	if err != nil {
		t.Fatalf("failed to hover: %v", err)
	}

	// Verify hover event was fired
	result, err := client.Eval(ctx, tabID, `window.hovered`)
	if err != nil {
		t.Fatalf("failed to verify hover: %v", err)
	}

	if result.Value != true {
		t.Errorf("expected hovered: true, got %v", result.Value)
	}
}

func TestClient_GetAttribute_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	_, err = client.Eval(ctx, tabID, `document.body.innerHTML = '<a id="link" href="https://example.com" data-value="42">Link</a>'`)
	if err != nil {
		t.Fatalf("failed to create element: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Get the href attribute
	value, err := client.GetAttribute(ctx, tabID, "#link", "href")
	if err != nil {
		t.Fatalf("failed to get attribute: %v", err)
	}

	if value != "https://example.com" {
		t.Errorf("expected href 'https://example.com', got '%s'", value)
	}

	// Get a data attribute
	dataValue, err := client.GetAttribute(ctx, tabID, "#link", "data-value")
	if err != nil {
		t.Fatalf("failed to get data attribute: %v", err)
	}

	if dataValue != "42" {
		t.Errorf("expected data-value '42', got '%s'", dataValue)
	}
}

func TestClient_GetAttribute_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Try to get attribute from non-existent element
	_, err = client.GetAttribute(ctx, tabID, "#nonexistent", "href")
	if err == nil {
		t.Error("expected error for non-existent element")
	}
}

func TestClient_Reload_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab and navigate
	tabID, err := client.NewTab(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(500 * time.Millisecond)

	// Reload the page
	err = client.Reload(ctx, tabID, false)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Verify we're still on example.com by checking the title
	result, err := client.Eval(ctx, tabID, `document.location.hostname`)
	if err != nil {
		t.Fatalf("failed to get hostname: %v", err)
	}

	if result.Value != "example.com" {
		t.Errorf("expected hostname 'example.com', got %v", result.Value)
	}
}

func TestClient_GoBack_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	// Navigate to second page
	_, err = client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate to example: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Go back
	err = client.GoBack(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to go back: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Verify we're back on about:blank
	result, err := client.Eval(ctx, tabID, `document.location.href`)
	if err != nil {
		t.Fatalf("failed to get href: %v", err)
	}

	if result.Value != "about:blank" {
		t.Errorf("expected href 'about:blank', got %v", result.Value)
	}
}

func TestClient_GetTitle_Success(t *testing.T) {
	// Uses isolated tab - not parallel due to Chrome resource contention
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create isolated tab for this test
	tabID, cleanup := createTestTab(t, client, ctx)
	defer cleanup()

	// Navigate to example.com which has a title
	_, err := client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(500 * time.Millisecond) // Give more time for page to load

	// Get the title
	title, err := client.GetTitle(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}

	if title == "" {
		t.Error("expected non-empty title")
	}
}

func TestClient_GetURL_Success(t *testing.T) {
	// Uses isolated tab - not parallel due to Chrome resource contention
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create isolated tab for this test
	tabID, cleanup := createTestTab(t, client, ctx)
	defer cleanup()

	// Navigate to example.com
	_, err := client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(500 * time.Millisecond) // Give more time for page to load

	// Get the URL
	url, err := client.GetURL(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to get URL: %v", err)
	}

	if !strings.Contains(url, "example.com") {
		t.Errorf("expected URL to contain 'example.com', got %s", url)
	}
}

func TestClient_NewTab_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Get initial tab count
	initialPages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	initialCount := len(initialPages)

	// Create new tab
	targetID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create new tab: %v", err)
	}

	if targetID == "" {
		t.Error("expected non-empty target ID")
	}

	// Verify tab count increased
	newPages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages after: %v", err)
	}

	if len(newPages) != initialCount+1 {
		t.Errorf("expected %d pages, got %d", initialCount+1, len(newPages))
	}

	// Clean up - close the new tab
	client.CloseTab(ctx, targetID)
}

func TestClient_DoubleClick_Success(t *testing.T) {
	// Uses isolated tab - not parallel due to Chrome resource contention
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create isolated tab for this test
	tabID, cleanup := createTestTab(t, client, ctx)
	defer cleanup()

	_, err := client.Eval(ctx, tabID, `
		document.body.innerHTML = '<button id="dbl-btn">Double Click Me</button>';
		window.dblClicked = false;
		document.getElementById('dbl-btn').addEventListener('dblclick', () => {
			window.dblClicked = true;
		});
	`)
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	err = client.DoubleClick(ctx, tabID, "#dbl-btn")
	if err != nil {
		t.Fatalf("failed to double-click: %v", err)
	}

	result, err := client.Eval(ctx, tabID, `window.dblClicked`)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	if result.Value != true {
		t.Errorf("expected dblClicked true, got %v", result.Value)
	}
}

func TestClient_Check_Success(t *testing.T) {
	// Uses isolated tab - not parallel due to Chrome resource contention
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create isolated tab for this test
	tabID, cleanup := createTestTab(t, client, ctx)
	defer cleanup()

	_, err := client.Eval(ctx, tabID, `document.body.innerHTML = '<input type="checkbox" id="cb" />'`)
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	err = client.Check(ctx, tabID, "#cb")
	if err != nil {
		t.Fatalf("failed to check: %v", err)
	}

	result, err := client.Eval(ctx, tabID, `document.querySelector('#cb').checked`)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	if result.Value != true {
		t.Errorf("expected checked true, got %v", result.Value)
	}
}

func TestClient_CountElements_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab with test content
	dataURL := `data:text/html,<html><body><div class="item">1</div><div class="item">2</div><div class="item">3</div></body></html>`
	tabID, err := client.NewTab(ctx, dataURL)
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(200 * time.Millisecond)

	count, err := client.CountElements(ctx, tabID, ".item")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 elements, got %d", count)
	}
}

func TestClient_SetViewport_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	err = client.SetViewport(ctx, tabID, 1024, 768)
	if err != nil {
		t.Fatalf("failed to set viewport: %v", err)
	}

	// Verify viewport via JS
	result, err := client.Eval(ctx, tabID, `window.innerWidth`)
	if err != nil {
		t.Fatalf("failed to get width: %v", err)
	}

	if result.Value != float64(1024) {
		t.Errorf("expected width 1024, got %v", result.Value)
	}
}

func TestClient_LocalStorage_Success(t *testing.T) {
	// Uses isolated tab - not parallel due to Chrome resource contention
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create isolated tab for this test
	tabID, cleanup := createTestTab(t, client, ctx)
	defer cleanup()

	// localStorage requires a real origin (not about:blank)
	_, err := client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(500 * time.Millisecond) // Give more time for page to fully load

	// Set a value
	err = client.SetLocalStorage(ctx, tabID, "testKey", "testValue")
	if err != nil {
		t.Fatalf("failed to set storage: %v", err)
	}

	// Get the value
	value, err := client.GetLocalStorage(ctx, tabID, "testKey")
	if err != nil {
		t.Fatalf("failed to get storage: %v", err)
	}

	if value != "testValue" {
		t.Errorf("expected 'testValue', got '%s'", value)
	}

	// Clear storage
	err = client.ClearLocalStorage(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to clear storage: %v", err)
	}

	// Verify cleared
	value, err = client.GetLocalStorage(ctx, tabID, "testKey")
	if err != nil {
		t.Fatalf("failed to get storage after clear: %v", err)
	}

	if value != "" {
		t.Errorf("expected empty after clear, got '%s'", value)
	}
}

func TestClient_RawCall_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Test browser-level command
	result, err := client.RawCall(ctx, "Target.getTargets", nil)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	// Verify it's valid JSON with targetInfos
	var resp struct {
		TargetInfos []interface{} `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Errorf("failed to parse result: %v", err)
	}
}

func TestClient_RawCallSession_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Test session-level command with params
	params := json.RawMessage(`{"expression":"1+1"}`)
	result, err := client.RawCallSession(ctx, tabID, "Runtime.evaluate", params)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	// Verify result contains value: 2
	var resp struct {
		Result struct {
			Value float64 `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Errorf("failed to parse result: %v", err)
	}
	if resp.Result.Value != 2 {
		t.Errorf("expected value 2, got %v", resp.Result.Value)
	}
}

func TestClient_Emulate_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Emulate iPhone 12 - just verify the method succeeds
	device := cdp.CommonDevices["iPhone 12"]
	err = client.Emulate(ctx, tabID, device)
	if err != nil {
		t.Fatalf("failed to emulate: %v", err)
	}

	// Note: In headless Chrome, the emulation may not affect window.innerWidth/Height
	// because there's no actual display. The emulation affects how the page renders
	// and what user-agent is reported. We just verify the CDP calls succeed.
}

func TestClient_EnableIntercept(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Enable interception
	config := cdp.InterceptConfig{
		URLPattern:        "*",
		InterceptResponse: true,
		Replacements:      map[string]string{"test": "replaced"},
	}
	err = client.EnableIntercept(ctx, tabID, config)
	if err != nil {
		t.Fatalf("failed to enable intercept: %v", err)
	}

	// Disable interception
	err = client.DisableIntercept(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to disable intercept: %v", err)
	}
}

func TestClient_InterceptModifyResponse(t *testing.T) {
	// Uses isolated tab - not parallel due to Chrome resource contention
	client := getSharedClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create isolated tab for this test
	tabID, cleanup := createTestTab(t, client, ctx)
	defer cleanup()

	// Enable interception - replace "Example Domain" with "PATCHED CONTENT"
	config := cdp.InterceptConfig{
		URLPattern:        "*",
		InterceptResponse: true,
		Replacements:      map[string]string{"Example Domain": "PATCHED CONTENT"},
	}
	err := client.EnableIntercept(ctx, tabID, config)
	if err != nil {
		t.Fatalf("failed to enable intercept: %v", err)
	}
	defer client.DisableIntercept(ctx, tabID) // Cleanup

	// Navigate to example.com
	_, err = client.Navigate(ctx, tabID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	// Wait for page load and interception
	time.Sleep(2 * time.Second)

	// Check if content was modified
	result, err := client.Eval(ctx, tabID, "document.body.innerText.includes('PATCHED')")
	if err != nil {
		t.Fatalf("failed to evaluate: %v", err)
	}

	patched, ok := result.Value.(bool)
	if !ok || !patched {
		t.Errorf("expected page content to be patched, got value=%v", result.Value)
	}
}

func TestClient_BlockURLs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create isolated tab
	tabID, err := client.NewTab(ctx, "about:blank")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)
	time.Sleep(100 * time.Millisecond)

	// Block URLs
	err = client.BlockURLs(ctx, tabID, []string{"*.js", "*.css"})
	if err != nil {
		t.Fatalf("failed to block URLs: %v", err)
	}

	// Unblock URLs
	err = client.UnblockURLs(ctx, tabID)
	if err != nil {
		t.Fatalf("failed to unblock URLs: %v", err)
	}
}
