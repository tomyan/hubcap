package chrome

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

// BridgeEvent represents an event from a bridge session.
type BridgeEvent struct {
	Type  string      `json:"type"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// Bridge is a persistent bidirectional message channel between the CLI and
// client-side JavaScript running in a browser tab.
type Bridge struct {
	Events    <-chan BridgeEvent
	id        string
	sessionID string
	client    *Client
	done      chan struct{}
	closeOnce sync.Once
}

// Close shuts down the bridge and cleans up resources.
func (b *Bridge) Close() {
	b.closeOnce.Do(func() {
		close(b.done)
	})
}

func generateBridgeID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return "__hubcap_bridge_" + hex.EncodeToString(b)
}

// StartBridge establishes a bidirectional message channel with client-side JS.
// The provided script receives `send` (function) and `messages` (async iterator)
// in scope. Events from the bridge (ready, message, error, closed) are delivered
// on the returned Bridge.Events channel.
func (c *Client) StartBridge(ctx context.Context, targetID string, script string) (*Bridge, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Runtime domain
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Runtime domain: %w", err)
	}

	id := generateBridgeID()
	bindingName := id + "_send"
	closedBindingName := id + "_closed"

	// Register bindings for JS→CLI communication
	_, err = c.CallSession(ctx, sessionID, "Runtime.addBinding", map[string]interface{}{
		"name": bindingName,
	})
	if err != nil {
		return nil, fmt.Errorf("adding send binding: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Runtime.addBinding", map[string]interface{}{
		"name": closedBindingName,
	})
	if err != nil {
		return nil, fmt.Errorf("adding closed binding: %w", err)
	}

	// Subscribe to binding called events
	bindingCh := c.subscribeEvent(sessionID, "Runtime.bindingCalled")

	events := make(chan BridgeEvent, 100)
	done := make(chan struct{})

	bridge := &Bridge{
		Events:    events,
		id:        id,
		sessionID: sessionID,
		client:    c,
		done:      done,
	}

	// Start goroutine to process binding events
	go func() {
		defer close(events)
		defer c.unsubscribeEvent(sessionID, "Runtime.bindingCalled", bindingCh)

		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				events <- BridgeEvent{Type: "closed", Error: "context cancelled"}
				return
			case raw, ok := <-bindingCh:
				if !ok {
					return
				}

				var binding struct {
					Name    string `json:"name"`
					Payload string `json:"payload"`
				}
				if err := json.Unmarshal(raw, &binding); err != nil {
					continue
				}

				if binding.Name == closedBindingName {
					var closedPayload struct {
						Reason string `json:"reason"`
					}
					json.Unmarshal([]byte(binding.Payload), &closedPayload)
					reason := closedPayload.Reason
					if reason == "" {
						reason = "script ended"
					}
					events <- BridgeEvent{Type: "closed", Data: reason}
					return
				}

				if binding.Name == bindingName {
					var data interface{}
					if err := json.Unmarshal([]byte(binding.Payload), &data); err != nil {
						events <- BridgeEvent{Type: "error", Error: fmt.Sprintf("invalid JSON from send(): %s", binding.Payload)}
						continue
					}
					events <- BridgeEvent{Type: "message", Data: data}
				}
			}
		}
	}()

	// Inject the bridge script
	wrappedScript := fmt.Sprintf(`
(async () => {
	const __id = %q;
	const __sendBinding = %q;
	const __closedBinding = %q;

	function send(data) {
		window[__sendBinding](JSON.stringify(data));
	}

	try {
		%s
	} catch (e) {
		window[__sendBinding](JSON.stringify({__bridge_error: true, message: e.message, stack: e.stack}));
	}

	window[__closedBinding](JSON.stringify({reason: "script ended"}));
})();
`, id, bindingName, closedBindingName, script)

	// Emit ready before running the script
	events <- BridgeEvent{Type: "ready"}

	// Run the script (fire and forget — it's async)
	_, err = c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    wrappedScript,
		"awaitPromise":  false,
		"returnByValue": false,
	})
	if err != nil {
		return nil, fmt.Errorf("injecting bridge script: %w", err)
	}

	return bridge, nil
}
