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

	// Create a page with an input via JS
	_, err = client.Eval(ctx, pages[0].ID, `
		document.body.innerHTML = '<input id="test-input" type="text" />';
	`)
	if err != nil {
		t.Fatalf("failed to create input: %v", err)
	}

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
