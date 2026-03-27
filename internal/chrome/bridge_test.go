package chrome_test

import (
	"context"
	"testing"
	"time"

	chrome "github.com/tomyan/hubcap/internal/chrome"
)

func TestBridge_SendFromJS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := chrome.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Given a page
	tabID, err := client.NewTabAndWait(ctx, "data:text/html,<html><body>bridge test</body></html>")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)

	// When we start a bridge that sends a message
	bridge, err := client.StartBridge(ctx, tabID, `send({greeting: "hello from JS"})`)
	if err != nil {
		t.Fatalf("failed to start bridge: %v", err)
	}
	defer bridge.Close()

	// Then we receive the ready event
	ev := <-bridge.Events
	if ev.Type != "ready" {
		t.Fatalf("expected ready event, got %s", ev.Type)
	}

	// And we receive the message
	ev = <-bridge.Events
	if ev.Type != "message" {
		t.Fatalf("expected message event, got %s", ev.Type)
	}

	data, ok := ev.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map data, got %T", ev.Data)
	}
	if data["greeting"] != "hello from JS" {
		t.Errorf("expected greeting 'hello from JS', got %v", data["greeting"])
	}
}

func TestBridge_MultipleMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := chrome.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Given a page
	tabID, err := client.NewTabAndWait(ctx, "data:text/html,<html><body>bridge test</body></html>")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)

	// When we start a bridge that sends multiple messages
	bridge, err := client.StartBridge(ctx, tabID, `
		send({n: 1});
		send({n: 2});
		send({n: 3});
	`)
	if err != nil {
		t.Fatalf("failed to start bridge: %v", err)
	}
	defer bridge.Close()

	// Then we receive ready + 3 messages
	ev := <-bridge.Events
	if ev.Type != "ready" {
		t.Fatalf("expected ready, got %s", ev.Type)
	}

	for i := 1; i <= 3; i++ {
		ev = <-bridge.Events
		if ev.Type != "message" {
			t.Fatalf("expected message, got %s", ev.Type)
		}
		data := ev.Data.(map[string]interface{})
		if int(data["n"].(float64)) != i {
			t.Errorf("expected n=%d, got %v", i, data["n"])
		}
	}
}

func TestBridge_ClosedOnScriptEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := chrome.Connect(ctx, "localhost", testChromePort)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Given a page
	tabID, err := client.NewTabAndWait(ctx, "data:text/html,<html><body>bridge test</body></html>")
	if err != nil {
		t.Fatalf("failed to create tab: %v", err)
	}
	defer client.CloseTab(ctx, tabID)

	// When we start a bridge with a script that finishes immediately
	bridge, err := client.StartBridge(ctx, tabID, `send("done")`)
	if err != nil {
		t.Fatalf("failed to start bridge: %v", err)
	}
	defer bridge.Close()

	// Then we get ready, message, and closed
	ev := <-bridge.Events
	if ev.Type != "ready" {
		t.Fatalf("expected ready, got %s", ev.Type)
	}

	ev = <-bridge.Events
	if ev.Type != "message" {
		t.Fatalf("expected message, got %s", ev.Type)
	}

	ev = <-bridge.Events
	if ev.Type != "closed" {
		t.Fatalf("expected closed, got %s", ev.Type)
	}
}
