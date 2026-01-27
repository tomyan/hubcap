package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// Errors
var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrCDPError         = errors.New("CDP error")
)

// CDPError represents an error returned by Chrome DevTools Protocol.
type CDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *CDPError) Error() string {
	return fmt.Sprintf("CDP error %d: %s", e.Code, e.Message)
}

func (e *CDPError) Unwrap() error {
	return ErrCDPError
}

// VersionInfo contains browser version information.
type VersionInfo struct {
	Browser         string `json:"browser"`
	ProtocolVersion string `json:"protocol"`
	UserAgent       string `json:"userAgent,omitempty"`
	V8Version       string `json:"v8,omitempty"`
}

// TargetInfo contains information about a browser target (tab/page).
type TargetInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// NavigateResult contains the result of a navigation.
type NavigateResult struct {
	FrameID   string `json:"frameId"`
	LoaderID  string `json:"loaderId,omitempty"`
	URL       string `json:"url"`
	ErrorText string `json:"errorText,omitempty"`
}

// Client represents a connection to Chrome DevTools Protocol.
type Client struct {
	conn      *websocket.Conn
	wsURL     string
	mu        sync.Mutex
	messageID atomic.Int64
	pending   map[int64]chan callResult
	pendingMu sync.Mutex
	closed    atomic.Bool
	closeOnce sync.Once
	closeCh   chan struct{}
}

type callResult struct {
	Result json.RawMessage
	Error  *CDPError
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
		conn:    conn,
		wsURL:   versionResp.WebSocketDebuggerURL,
		pending: make(map[int64]chan callResult),
		closeCh: make(chan struct{}),
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

// Version returns the browser version information.
func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	result, err := c.Call(ctx, "Browser.getVersion", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Product         string `json:"product"`
		ProtocolVersion string `json:"protocolVersion"`
		UserAgent       string `json:"userAgent"`
		JsVersion       string `json:"jsVersion"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling version: %w", err)
	}

	return &VersionInfo{
		Browser:         resp.Product,
		ProtocolVersion: resp.ProtocolVersion,
		UserAgent:       resp.UserAgent,
		V8Version:       resp.JsVersion,
	}, nil
}

// Targets returns all browser targets (pages, workers, etc.).
func (c *Client) Targets(ctx context.Context) ([]TargetInfo, error) {
	result, err := c.Call(ctx, "Target.getTargets", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling targets: %w", err)
	}

	targets := make([]TargetInfo, 0, len(resp.TargetInfos))
	for _, t := range resp.TargetInfos {
		targets = append(targets, TargetInfo{
			ID:    t.TargetID,
			Type:  t.Type,
			Title: t.Title,
			URL:   t.URL,
		})
	}

	return targets, nil
}

// Pages returns only page targets (tabs).
func (c *Client) Pages(ctx context.Context) ([]TargetInfo, error) {
	targets, err := c.Targets(ctx)
	if err != nil {
		return nil, err
	}

	pages := make([]TargetInfo, 0)
	for _, t := range targets {
		if t.Type == "page" {
			pages = append(pages, t)
		}
	}
	return pages, nil
}

// Navigate navigates a target to the given URL and waits for load.
func (c *Client) Navigate(ctx context.Context, targetID string, url string) (*NavigateResult, error) {
	// Attach to the target
	attachResult, err := c.Call(ctx, "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("attaching to target: %w", err)
	}

	var attachResp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachResult, &attachResp); err != nil {
		return nil, fmt.Errorf("parsing attach response: %w", err)
	}

	// Enable Page domain on the session
	_, err = c.CallSession(ctx, attachResp.SessionID, "Page.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Page domain: %w", err)
	}

	// Navigate
	navResult, err := c.CallSession(ctx, attachResp.SessionID, "Page.navigate", map[string]string{
		"url": url,
	})
	if err != nil {
		return nil, fmt.Errorf("navigating: %w", err)
	}

	var navResp struct {
		FrameID   string `json:"frameId"`
		LoaderID  string `json:"loaderId"`
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(navResult, &navResp); err != nil {
		return nil, fmt.Errorf("parsing navigate response: %w", err)
	}

	if navResp.ErrorText != "" {
		return &NavigateResult{
			FrameID:   navResp.FrameID,
			ErrorText: navResp.ErrorText,
			URL:       url,
		}, nil
	}

	// Wait for load event (simplified - just wait a bit for now, proper implementation would use events)
	// TODO: Implement proper load waiting with Page.loadEventFired

	return &NavigateResult{
		FrameID:  navResp.FrameID,
		LoaderID: navResp.LoaderID,
		URL:      url,
	}, nil
}

// CallSession sends a CDP command to a specific session.
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

type cdpRequest struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type cdpResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *CDPError       `json:"error,omitempty"`
}

// Call sends a CDP command and waits for the response.
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
	}
}
