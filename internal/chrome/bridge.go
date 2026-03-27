package chrome

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
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

// CloseIterator signals the JS async iterator to close, allowing the script
// to finish cleanly. The bridge remains open to receive the final closed event.
func (b *Bridge) CloseIterator(ctx context.Context) error {
	closeExpr := fmt.Sprintf(`window[%q] && window[%q]()`, b.id+"_close", b.id+"_close")
	_, err := b.client.CallSession(ctx, b.sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    closeExpr,
		"returnByValue": true,
	})
	if err != nil {
		return fmt.Errorf("closing bridge iterator: %w", err)
	}
	return nil
}

// Send delivers a message to the JS async iterator.
func (b *Bridge) Send(ctx context.Context, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshalling message: %w", err)
	}

	pushExpr := fmt.Sprintf(`window[%q](%s)`, b.id+"_push", string(jsonBytes))
	_, err = b.client.CallSession(ctx, b.sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    pushExpr,
		"returnByValue": true,
	})
	if err != nil {
		return fmt.Errorf("pushing message to bridge: %w", err)
	}
	return nil
}

func generateBridgeID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return "__hubcap_bridge_" + hex.EncodeToString(b)
}

// bridgeScript returns the JS that sets up the async iterator, send function,
// keepalive watchdog, and wraps the user's script.
func bridgeScript(id, bindingName, closedBindingName, userScript string) string {
	return fmt.Sprintf(`
(async () => {
	const __sendBinding = %q;
	const __closedBinding = %q;
	const __pushName = %q;
	const __heartbeatName = %q;
	const __closeName = %q;

	// Async iterator backed by a promise queue
	const __buffer = [];
	const __waiters = [];
	let __closed = false;

	window[__pushName] = function(data) {
		if (__closed) return;
		if (__waiters.length > 0) {
			__waiters.shift().resolve({value: data, done: false});
		} else {
			__buffer.push(data);
		}
	};

	function __closePush() {
		__closed = true;
		for (const w of __waiters) {
			w.resolve({value: undefined, done: true});
		}
		__waiters.length = 0;
	}

	// Keepalive watchdog — close iterator if no heartbeat for 6 seconds
	let __lastHeartbeat = Date.now();
	window[__heartbeatName] = function() {
		__lastHeartbeat = Date.now();
	};

	// Close function — called from CLI to gracefully shut down the iterator
	window[__closeName] = function() {
		__closePush();
	};
	const __watchdog = setInterval(() => {
		if (Date.now() - __lastHeartbeat > 6000) {
			clearInterval(__watchdog);
			__closePush();
		}
	}, 1000);

	const messages = {
		[Symbol.asyncIterator]() { return this; },
		next() {
			if (__buffer.length > 0) {
				return Promise.resolve({value: __buffer.shift(), done: false});
			}
			if (__closed) {
				return Promise.resolve({value: undefined, done: true});
			}
			return new Promise(resolve => __waiters.push({resolve}));
		},
		return() {
			__closePush();
			return Promise.resolve({value: undefined, done: true});
		}
	};

	function send(data) {
		window[__sendBinding](JSON.stringify(data));
	}

	try {
		%s
	} catch (e) {
		window[__sendBinding](JSON.stringify({__bridge_error: true, message: e.message, stack: e.stack}));
	}

	clearInterval(__watchdog);
	__closePush();
	window[__closedBinding](JSON.stringify({reason: "script ended"}));
})();
`, bindingName, closedBindingName, id+"_push", id+"_heartbeat", id+"_close", userScript)
}

// sendHeartbeats sends periodic heartbeats to the JS watchdog.
// Returns when done channel is closed or context is cancelled.
func (b *Bridge) sendHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	heartbeatExpr := fmt.Sprintf(`window[%q] && window[%q]()`, b.id+"_heartbeat", b.id+"_heartbeat")
	for {
		select {
		case <-b.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.client.CallSession(ctx, b.sessionID, "Runtime.evaluate", map[string]interface{}{
				"expression":    heartbeatExpr,
				"returnByValue": true,
			})
		}
	}
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

					// Check for bridge error wrapper
					if m, ok := data.(map[string]interface{}); ok {
						if _, isErr := m["__bridge_error"]; isErr {
							msg, _ := m["message"].(string)
							events <- BridgeEvent{Type: "error", Error: msg}
							continue
						}
					}

					events <- BridgeEvent{Type: "message", Data: data}
				}
			}
		}
	}()

	// Start keepalive heartbeats
	go bridge.sendHeartbeats(ctx)

	// Emit ready before running the script
	events <- BridgeEvent{Type: "ready"}

	// Run the script (fire and forget — it's async)
	wrappedScript := bridgeScript(id, bindingName, closedBindingName, script)
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
