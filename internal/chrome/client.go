package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a Chrome DevTools Protocol client.
type Client struct {
	conn            *websocket.Conn
	wsURL           string
	mu              sync.Mutex
	messageID       atomic.Int64
	pending         map[int64]chan callResult
	pendingMu       sync.Mutex
	eventHandlers   map[string][]chan json.RawMessage // key: "sessionID:method"
	eventHandlersMu sync.Mutex
	sessions        map[string]string // targetID -> sessionID (session cache)
	sessionsMu      sync.Mutex
	closed          atomic.Bool
	closeOnce       sync.Once
	closeCh         chan struct{}
}

type callResult struct {
	Result json.RawMessage
	Error  *ProtocolError
}

// Connect establishes a connection to Chrome at the given host and port.
func Connect(ctx context.Context, host string, port int) (*Client, error) {
	// First, get the WebSocket URL from the JSON endpoint
	jsonURL := fmt.Sprintf("http://%s:%d/json/version", host, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jsonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to Chrome: %w", err)
	}
	defer resp.Body.Close()

	var versionResp struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		return nil, fmt.Errorf("decoding version response: %w", err)
	}

	if versionResp.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("no WebSocket URL in response")
	}

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, versionResp.WebSocketDebuggerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to WebSocket: %w", err)
	}

	client := &Client{
		conn:          conn,
		wsURL:         versionResp.WebSocketDebuggerURL,
		pending:       make(map[int64]chan callResult),
		eventHandlers: make(map[string][]chan json.RawMessage),
		sessions:      make(map[string]string),
		closeCh:       make(chan struct{}),
	}

	// Start message reader
	go client.readMessages()

	return client, nil
}

// WebSocketURL returns the WebSocket URL used for this connection.
func (c *Client) WebSocketURL() string {
	return c.wsURL
}

// Close closes the connection to Chrome.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		// Detach all cached sessions (best effort)
		c.sessionsMu.Lock()
		sessions := make(map[string]string)
		for k, v := range c.sessions {
			sessions[k] = v
		}
		c.sessions = make(map[string]string)
		c.sessionsMu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for _, sessionID := range sessions {
			c.Call(ctx, "Target.detachFromTarget", map[string]interface{}{
				"sessionId": sessionID,
			})
		}

		c.closed.Store(true)
		close(c.closeCh)
		err = c.conn.Close()

		// Wake up all pending callers
		c.pendingMu.Lock()
		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = make(map[int64]chan callResult)
		c.pendingMu.Unlock()
	})
	return err
}

func (c *Client) attachToTarget(ctx context.Context, targetID string) (string, error) {
	// Check cache first
	c.sessionsMu.Lock()
	if sessionID, ok := c.sessions[targetID]; ok {
		c.sessionsMu.Unlock()
		return sessionID, nil
	}
	c.sessionsMu.Unlock()

	// Create new session
	attachResult, err := c.Call(ctx, "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	if err != nil {
		return "", fmt.Errorf("attaching to target: %w", err)
	}

	var attachResp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachResult, &attachResp); err != nil {
		return "", fmt.Errorf("parsing attach response: %w", err)
	}

	// Cache the session
	c.sessionsMu.Lock()
	c.sessions[targetID] = attachResp.SessionID
	c.sessionsMu.Unlock()

	return attachResp.SessionID, nil
}

type cdpRequest struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type cdpResponse struct {
	ID        int64           `json:"id"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *ProtocolError  `json:"error,omitempty"`
	Method    string          `json:"method,omitempty"`    // For events
	Params    json.RawMessage `json:"params,omitempty"`    // For events
	SessionID string          `json:"sessionId,omitempty"` // For session events
}

// Call sends a protocol command and waits for the response.
func (c *Client) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, ErrConnectionClosed
	}

	id := c.messageID.Add(1)

	req := cdpRequest{
		ID:     id,
		Method: method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshaling params: %w", err)
		}
		req.Params = data
	}

	// Create response channel
	respChan := make(chan callResult, 1)
	c.pendingMu.Lock()
	c.pending[id] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	// Send message
	c.mu.Lock()
	err := c.conn.WriteJSON(req)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("sending message: %w", err)
	}

	// Wait for response
	select {
	case result, ok := <-respChan:
		if !ok {
			return nil, ErrConnectionClosed
		}
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Result, nil
	case <-c.closeCh:
		return nil, ErrConnectionClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CallSession sends a protocol command to a specific session and waits for the response.
func (c *Client) CallSession(ctx context.Context, sessionID string, method string, params interface{}) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, ErrConnectionClosed
	}

	id := c.messageID.Add(1)

	type sessionRequest struct {
		ID        int64           `json:"id"`
		SessionID string          `json:"sessionId"`
		Method    string          `json:"method"`
		Params    json.RawMessage `json:"params,omitempty"`
	}

	req := sessionRequest{
		ID:        id,
		SessionID: sessionID,
		Method:    method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshaling params: %w", err)
		}
		req.Params = data
	}

	// Create response channel
	respChan := make(chan callResult, 1)
	c.pendingMu.Lock()
	c.pending[id] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	// Send message
	c.mu.Lock()
	err := c.conn.WriteJSON(req)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("sending message: %w", err)
	}

	// Wait for response
	select {
	case result, ok := <-respChan:
		if !ok {
			return nil, ErrConnectionClosed
		}
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Result, nil
	case <-c.closeCh:
		return nil, ErrConnectionClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) readMessages() {
	defer c.Close()

	for {
		var resp cdpResponse
		if err := c.conn.ReadJSON(&resp); err != nil {
			return
		}

		// Route response to waiting caller
		if resp.ID > 0 {
			c.pendingMu.Lock()
			if ch, ok := c.pending[resp.ID]; ok {
				ch <- callResult{
					Result: resp.Result,
					Error:  resp.Error,
				}
			}
			c.pendingMu.Unlock()
		}

		// Route events to handlers
		if resp.Method != "" {
			key := resp.SessionID + ":" + resp.Method
			c.eventHandlersMu.Lock()
			handlers := c.eventHandlers[key]
			for _, h := range handlers {
				select {
				case h <- resp.Params:
				default:
					// Drop if channel is full
				}
			}
			c.eventHandlersMu.Unlock()
		}
	}
}

// subscribeEvent registers a handler for protocol events.
func (c *Client) subscribeEvent(sessionID, method string) chan json.RawMessage {
	ch := make(chan json.RawMessage, 100)
	key := sessionID + ":" + method

	c.eventHandlersMu.Lock()
	c.eventHandlers[key] = append(c.eventHandlers[key], ch)
	c.eventHandlersMu.Unlock()

	return ch
}

// unsubscribeEvent removes an event handler.
func (c *Client) unsubscribeEvent(sessionID, method string, ch chan json.RawMessage) {
	key := sessionID + ":" + method

	c.eventHandlersMu.Lock()
	defer c.eventHandlersMu.Unlock()

	handlers := c.eventHandlers[key]
	for i, h := range handlers {
		if h == ch {
			c.eventHandlers[key] = append(handlers[:i], handlers[i+1:]...)
			close(ch)
			return
		}
	}
}

// RawCall sends a raw protocol command with JSON params.
func (c *Client) RawCall(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	var p interface{}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params JSON: %w", err)
		}
	}
	return c.Call(ctx, method, p)
}

// RawCallSession sends a raw protocol command to a specific target/session.
func (c *Client) RawCallSession(ctx context.Context, targetID string, method string, params json.RawMessage) (json.RawMessage, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	var p interface{}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params JSON: %w", err)
		}
	}
	return c.CallSession(ctx, sessionID, method, p)
}
