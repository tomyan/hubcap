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
