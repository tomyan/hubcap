package cdp_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tomyan/cdp-cli/internal/cdp"
)

func TestClient_Version_ReturnsVersionInfo(t *testing.T) {
	// Skip if no Chrome available (integration test)
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Port 1 should fail to connect
	_, err := cdp.Connect(ctx, "localhost", 1)
	if err == nil {
		t.Error("expected connection to fail on port 1")
	}
}

func TestClient_Connect_FailsWithBadHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := cdp.Connect(ctx, "nonexistent.invalid", 9222)
	if err == nil {
		t.Error("expected connection to fail with invalid host")
	}
}

func TestClient_WebSocketURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	wsURL := client.WebSocketURL()
	if wsURL == "" {
		t.Error("expected non-empty WebSocket URL")
	}
	if !strings.HasPrefix(wsURL, "ws://") {
		t.Errorf("expected WebSocket URL to start with ws://, got %s", wsURL)
	}
}

func TestClient_Call_ReturnsErrorOnClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Call invalid method
	_, err = client.Call(ctx, "Invalid.nonExistentMethod", nil)
	if err == nil {
		t.Error("expected error for invalid method")
	}

	// Should be a CDP error
	if !errors.Is(err, cdp.ErrCDPError) {
		t.Errorf("expected ErrCDPError, got %v", err)
	}
}

func TestClient_Targets_ReturnsPages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Get a page to navigate
	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	result, err := client.Navigate(ctx, pages[0].ID, "https://example.com")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to invalid URL - should still work but result in error page
	result, err := client.Navigate(ctx, pages[0].ID, "not-a-valid-url")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	result, err := client.Navigate(ctx, pages[0].ID, "https://example.com")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	data, err := client.Screenshot(ctx, pages[0].ID, cdp.ScreenshotOptions{
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	data, err := client.Screenshot(ctx, pages[0].ID, cdp.ScreenshotOptions{
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	result, err := client.Eval(ctx, pages[0].ID, "1 + 2")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	result, err := client.Eval(ctx, pages[0].ID, "'hello' + ' world'")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	result, err := client.Eval(ctx, pages[0].ID, "({a: 1, b: 'test'})")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	_, err = client.Eval(ctx, pages[0].ID, "throw new Error('test error')")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to example.com which has a body element
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	// Query for body element
	result, err := client.Query(ctx, pages[0].ID, "body")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Query for non-existent element
	result, err := client.Query(ctx, pages[0].ID, "#nonexistent-element-12345")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	result, err := client.Query(ctx, pages[0].ID, "body")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to example.com which has a link
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	// Click the link (example.com has "More information..." link)
	err = client.Click(ctx, pages[0].ID, "a")
	if err != nil {
		t.Fatalf("failed to click: %v", err)
	}

	// If we got here without error, click succeeded
}

func TestClient_Click_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state from previous tests
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Try to click non-existent element
	err = client.Click(ctx, pages[0].ID, "#nonexistent-element-12345")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}

	// Small delay to let page settle
	time.Sleep(50 * time.Millisecond)

	// Create a page with an input via JS
	_, err = client.Eval(ctx, pages[0].ID, `
		document.body.innerHTML = '<input id="test-input" type="text" />';
	`)
	if err != nil {
		t.Fatalf("failed to create input: %v", err)
	}

	// Small delay to let DOM settle
	time.Sleep(50 * time.Millisecond)

	// Fill the input
	err = client.Fill(ctx, pages[0].ID, "#test-input", "hello world")
	if err != nil {
		t.Fatalf("failed to fill: %v", err)
	}

	// Verify the value
	result, err := client.Eval(ctx, pages[0].ID, `document.querySelector('#test-input').value`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Try to fill non-existent element
	err = client.Fill(ctx, pages[0].ID, "#nonexistent-input-12345", "test")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Create a test element
	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<div id="test"><span>Hello</span></div>'`)
	if err != nil {
		t.Fatalf("failed to create element: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Get HTML
	html, err := client.GetHTML(ctx, pages[0].ID, "#test")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	_, err = client.GetHTML(ctx, pages[0].ID, "#nonexistent-12345")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Create element immediately
	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<div id="exists">Test</div>'`)
	if err != nil {
		t.Fatalf("failed to create element: %v", err)
	}

	// Wait should return immediately
	err = client.WaitFor(ctx, pages[0].ID, "#exists", 5*time.Second)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Set up delayed element creation (500ms)
	_, err = client.Eval(ctx, pages[0].ID, `
		document.body.innerHTML = '';
		setTimeout(() => {
			document.body.innerHTML = '<div id="delayed">Appeared</div>';
		}, 500);
	`)
	if err != nil {
		t.Fatalf("failed to set up delayed element: %v", err)
	}

	// Wait should poll and find it
	err = client.WaitFor(ctx, pages[0].ID, "#delayed", 5*time.Second)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Wait for non-existent element with short timeout
	err = client.WaitFor(ctx, pages[0].ID, "#never-exists", 500*time.Millisecond)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Create element with text
	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<div id="test">Hello <span>World</span>!</div>'`)
	if err != nil {
		t.Fatalf("failed to create element: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	text, err := client.GetText(ctx, pages[0].ID, "#test")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset DOM state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Create a page with an input and keydown counter
	_, err = client.Eval(ctx, pages[0].ID, `
		document.body.innerHTML = '<input id="test-input" type="text" />';
		window.keydownCount = 0;
		document.querySelector('#test-input').addEventListener('keydown', () => { window.keydownCount++; });
	`)
	if err != nil {
		t.Fatalf("failed to create input: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Focus the input first
	_, err = client.Eval(ctx, pages[0].ID, `document.querySelector('#test-input').focus()`)
	if err != nil {
		t.Fatalf("failed to focus input: %v", err)
	}

	// Type "abc" character by character
	err = client.Type(ctx, pages[0].ID, "abc")
	if err != nil {
		t.Fatalf("failed to type: %v", err)
	}

	// Verify the value
	result, err := client.Eval(ctx, pages[0].ID, `document.querySelector('#test-input').value`)
	if err != nil {
		t.Fatalf("failed to verify value: %v", err)
	}

	if result.Value != "abc" {
		t.Errorf("expected 'abc', got %v", result.Value)
	}

	// Verify keydown events were fired (should be 3, one per character)
	countResult, err := client.Eval(ctx, pages[0].ID, `window.keydownCount`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page to reset state
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Start capturing console messages
	messages, err := client.CaptureConsole(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("failed to start console capture: %v", err)
	}

	// Trigger some console messages via eval
	_, err = client.Eval(ctx, pages[0].ID, `
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to a page to have cookies
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Get cookies - should return a slice (may be empty)
	cookies, err := client.GetCookies(ctx, pages[0].ID)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to a page first
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Set a cookie
	err = client.SetCookie(ctx, pages[0].ID, cdp.Cookie{
		Name:   "test_cookie",
		Value:  "test_value",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie: %v", err)
	}

	// Verify cookie was set
	cookies, err := client.GetCookies(ctx, pages[0].ID)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to a page with content
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Generate PDF
	data, err := client.PrintToPDF(ctx, pages[0].ID, cdp.PDFOptions{})
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to a page first
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Set a cookie first
	err = client.SetCookie(ctx, pages[0].ID, cdp.Cookie{
		Name:   "delete_test",
		Value:  "test_value",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie: %v", err)
	}

	// Verify cookie exists
	cookies, err := client.GetCookies(ctx, pages[0].ID)
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
	err = client.DeleteCookie(ctx, pages[0].ID, "delete_test", "example.com")
	if err != nil {
		t.Fatalf("failed to delete cookie: %v", err)
	}

	// Verify cookie is gone
	cookies, err = client.GetCookies(ctx, pages[0].ID)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to a page first
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Set some cookies
	err = client.SetCookie(ctx, pages[0].ID, cdp.Cookie{
		Name:   "clear_test1",
		Value:  "value1",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie 1: %v", err)
	}
	err = client.SetCookie(ctx, pages[0].ID, cdp.Cookie{
		Name:   "clear_test2",
		Value:  "value2",
		Domain: "example.com",
	})
	if err != nil {
		t.Fatalf("failed to set cookie 2: %v", err)
	}

	// Clear all cookies
	err = client.ClearCookies(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("failed to clear cookies: %v", err)
	}

	// Verify cookies are gone
	cookies, err := client.GetCookies(ctx, pages[0].ID)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page and create an input
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<input id="focus-test" type="text" />'`)
	if err != nil {
		t.Fatalf("failed to create input: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Focus the element
	err = client.Focus(ctx, pages[0].ID, "#focus-test")
	if err != nil {
		t.Fatalf("failed to focus: %v", err)
	}

	// Verify focus via JS
	result, err := client.Eval(ctx, pages[0].ID, `document.activeElement.id`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page first
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate to blank: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Start capturing network events
	events, err := client.CaptureNetwork(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("failed to start network capture: %v", err)
	}

	// Navigate to a page to trigger network requests
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Navigate(ctx, pages[0].ID, "https://example.com")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page and create a form with input
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Create a form with a text input that tracks key events
	_, err = client.Eval(ctx, pages[0].ID, `
		document.body.innerHTML = '<input id="test-input" type="text" />';
		window.lastKey = '';
		document.getElementById('test-input').addEventListener('keydown', (e) => {
			window.lastKey = e.key;
		});
	`)
	if err != nil {
		t.Fatalf("failed to create input: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Focus the input
	err = client.Focus(ctx, pages[0].ID, "#test-input")
	if err != nil {
		t.Fatalf("failed to focus: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Press the Enter key
	err = client.PressKey(ctx, pages[0].ID, "Enter")
	if err != nil {
		t.Fatalf("failed to press key: %v", err)
	}

	// Verify the key was pressed
	result, err := client.Eval(ctx, pages[0].ID, `window.lastKey`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page and create a button that tracks hover
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = client.Eval(ctx, pages[0].ID, `
		document.body.innerHTML = '<button id="hover-btn" style="width:100px;height:50px;">Hover me</button>';
		window.hovered = false;
		document.getElementById('hover-btn').addEventListener('mouseenter', () => {
			window.hovered = true;
		});
	`)
	if err != nil {
		t.Fatalf("failed to create button: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Hover over the button
	err = client.Hover(ctx, pages[0].ID, "#hover-btn")
	if err != nil {
		t.Fatalf("failed to hover: %v", err)
	}

	// Verify hover event was fired
	result, err := client.Eval(ctx, pages[0].ID, `window.hovered`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page and create an element with attributes
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<a id="link" href="https://example.com" data-value="42">Link</a>'`)
	if err != nil {
		t.Fatalf("failed to create element: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Get the href attribute
	value, err := client.GetAttribute(ctx, pages[0].ID, "#link", "href")
	if err != nil {
		t.Fatalf("failed to get attribute: %v", err)
	}

	if value != "https://example.com" {
		t.Errorf("expected href 'https://example.com', got '%s'", value)
	}

	// Get a data attribute
	dataValue, err := client.GetAttribute(ctx, pages[0].ID, "#link", "data-value")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to blank page
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Try to get attribute from non-existent element
	_, err = client.GetAttribute(ctx, pages[0].ID, "#nonexistent", "href")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to example.com first
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Reload the page
	err = client.Reload(ctx, pages[0].ID, false)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Verify we're still on example.com by checking the title
	result, err := client.Eval(ctx, pages[0].ID, `document.location.hostname`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to first page
	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate to blank: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Navigate to second page
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate to example: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Go back
	err = client.GoBack(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("failed to go back: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Verify we're back on about:blank
	result, err := client.Eval(ctx, pages[0].ID, `document.location.href`)
	if err != nil {
		t.Fatalf("failed to get href: %v", err)
	}

	if result.Value != "about:blank" {
		t.Errorf("expected href 'about:blank', got %v", result.Value)
	}
}

func TestClient_GetTitle_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to example.com which has a title
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Get the title
	title, err := client.GetTitle(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}

	if title == "" {
		t.Error("expected non-empty title")
	}
}

func TestClient_GetURL_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Navigate to example.com
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Get the URL
	url, err := client.GetURL(ctx, pages[0].ID)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = client.Eval(ctx, pages[0].ID, `
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

	err = client.DoubleClick(ctx, pages[0].ID, "#dbl-btn")
	if err != nil {
		t.Fatalf("failed to double-click: %v", err)
	}

	result, err := client.Eval(ctx, pages[0].ID, `window.dblClicked`)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	if result.Value != true {
		t.Errorf("expected dblClicked true, got %v", result.Value)
	}
}

func TestClient_Check_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<input type="checkbox" id="cb" />'`)
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	err = client.Check(ctx, pages[0].ID, "#cb")
	if err != nil {
		t.Fatalf("failed to check: %v", err)
	}

	result, err := client.Eval(ctx, pages[0].ID, `document.querySelector('#cb').checked`)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	_, err = client.Navigate(ctx, pages[0].ID, "about:blank")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = client.Eval(ctx, pages[0].ID, `document.body.innerHTML = '<div class="item">1</div><div class="item">2</div><div class="item">3</div>'`)
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	count, err := client.CountElements(ctx, pages[0].ID, ".item")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	err = client.SetViewport(ctx, pages[0].ID, 1024, 768)
	if err != nil {
		t.Fatalf("failed to set viewport: %v", err)
	}

	// Verify viewport via JS
	result, err := client.Eval(ctx, pages[0].ID, `window.innerWidth`)
	if err != nil {
		t.Fatalf("failed to get width: %v", err)
	}

	if result.Value != float64(1024) {
		t.Errorf("expected width 1024, got %v", result.Value)
	}
}

func TestClient_LocalStorage_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// localStorage requires a real origin (not about:blank)
	_, err = client.Navigate(ctx, pages[0].ID, "https://example.com")
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Set a value
	err = client.SetLocalStorage(ctx, pages[0].ID, "testKey", "testValue")
	if err != nil {
		t.Fatalf("failed to set storage: %v", err)
	}

	// Get the value
	value, err := client.GetLocalStorage(ctx, pages[0].ID, "testKey")
	if err != nil {
		t.Fatalf("failed to get storage: %v", err)
	}

	if value != "testValue" {
		t.Errorf("expected 'testValue', got '%s'", value)
	}

	// Clear storage
	err = client.ClearLocalStorage(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("failed to clear storage: %v", err)
	}

	// Verify cleared
	value, err = client.GetLocalStorage(ctx, pages[0].ID, "testKey")
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
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

	client, err := cdp.Connect(ctx, "localhost", 9222)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	pages, err := client.Pages(ctx)
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no pages available")
	}

	// Test session-level command with params
	params := json.RawMessage(`{"expression":"1+1"}`)
	result, err := client.RawCallSession(ctx, pages[0].ID, "Runtime.evaluate", params)
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
