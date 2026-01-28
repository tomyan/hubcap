package cdp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// ScreenshotOptions configures screenshot capture.
type ScreenshotOptions struct {
	Format  string // "png", "jpeg", "webp"
	Quality int    // 0-100, only for jpeg/webp
}

// ScreenshotResult contains metadata about a captured screenshot.
type ScreenshotResult struct {
	Format string `json:"format"`
	Size   int    `json:"size"`
}

// EvalResult contains the result of evaluating a JavaScript expression.
type EvalResult struct {
	Value interface{} `json:"value"`
	Type  string      `json:"type,omitempty"`
}

// ConsoleMessage represents a console message from the browser.
type ConsoleMessage struct {
	Type string `json:"type"` // "log", "warn", "error", "info", "debug"
	Text string `json:"text"`
}

// NetworkEvent represents a network request or response event.
type NetworkEvent struct {
	Type      string `json:"type"`      // "request" or "response"
	RequestID string `json:"requestId"` // unique identifier for matching request/response
	URL       string `json:"url"`
	Method    string `json:"method,omitempty"`    // HTTP method (requests only)
	Status    int    `json:"status,omitempty"`    // HTTP status code (responses only)
	MimeType  string `json:"mimeType,omitempty"`  // MIME type (responses only)
}

// Cookie represents a browser cookie.
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain,omitempty"`
	Path     string  `json:"path,omitempty"`
	Expires  float64 `json:"expires,omitempty"`
	HTTPOnly bool    `json:"httpOnly,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	SameSite string  `json:"sameSite,omitempty"`
}

// PDFOptions configures PDF generation.
type PDFOptions struct {
	Landscape           bool    `json:"landscape,omitempty"`
	PrintBackground     bool    `json:"printBackground,omitempty"`
	Scale               float64 `json:"scale,omitempty"`
	PaperWidth          float64 `json:"paperWidth,omitempty"`  // inches
	PaperHeight         float64 `json:"paperHeight,omitempty"` // inches
	MarginTop           float64 `json:"marginTop,omitempty"`   // inches
	MarginBottom        float64 `json:"marginBottom,omitempty"`
	MarginLeft          float64 `json:"marginLeft,omitempty"`
	MarginRight         float64 `json:"marginRight,omitempty"`
	PageRanges          string  `json:"pageRanges,omitempty"` // e.g. "1-5, 8"
	PreferCSSPageSize   bool    `json:"preferCSSPageSize,omitempty"`
}

// QueryResult contains the result of querying for a DOM element.
type QueryResult struct {
	NodeID     int               `json:"nodeId"`
	TagName    string            `json:"tagName,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Client represents a connection to Chrome DevTools Protocol.
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

// attachToTarget attaches to a target and returns the session ID.
// Sessions are cached and reused to avoid creating too many sessions.
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

// Navigate navigates a target to the given URL and waits for load.
func (c *Client) Navigate(ctx context.Context, targetID string, url string) (*NavigateResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Page domain on the session
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Page domain: %w", err)
	}

	// Navigate
	navResult, err := c.CallSession(ctx, sessionID, "Page.navigate", map[string]string{
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

// NavigateAndWait navigates to a URL and waits for the page load event.
func (c *Client) NavigateAndWait(ctx context.Context, targetID string, url string) (*NavigateResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Page domain on the session
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Page domain: %w", err)
	}

	// Subscribe to load event before navigating
	loadCh := c.subscribeEvent(sessionID, "Page.loadEventFired")
	defer c.unsubscribeEvent(sessionID, "Page.loadEventFired", loadCh)

	// Navigate
	navResult, err := c.CallSession(ctx, sessionID, "Page.navigate", map[string]string{
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

	// Wait for load event with timeout
	select {
	case <-loadCh:
		// Load completed
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for page load")
	}

	return &NavigateResult{
		FrameID:  navResp.FrameID,
		LoaderID: navResp.LoaderID,
		URL:      url,
	}, nil
}

// GoBack navigates back in history.
func (c *Client) GoBack(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Page domain
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Page domain: %w", err)
	}

	// Get navigation history to check if we can go back
	histResult, err := c.CallSession(ctx, sessionID, "Page.getNavigationHistory", nil)
	if err != nil {
		return fmt.Errorf("getting navigation history: %w", err)
	}

	var histResp struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(histResult, &histResp); err != nil {
		return fmt.Errorf("parsing navigation history: %w", err)
	}

	if histResp.CurrentIndex == 0 {
		return fmt.Errorf("no history to go back to")
	}

	// Navigate to previous entry
	prevEntry := histResp.Entries[histResp.CurrentIndex-1]
	_, err = c.CallSession(ctx, sessionID, "Page.navigateToHistoryEntry", map[string]interface{}{
		"entryId": prevEntry.ID,
	})
	if err != nil {
		return fmt.Errorf("navigating to history entry: %w", err)
	}

	return nil
}

// GoForward navigates forward in history.
func (c *Client) GoForward(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Page domain
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Page domain: %w", err)
	}

	// Get navigation history
	histResult, err := c.CallSession(ctx, sessionID, "Page.getNavigationHistory", nil)
	if err != nil {
		return fmt.Errorf("getting navigation history: %w", err)
	}

	var histResp struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(histResult, &histResp); err != nil {
		return fmt.Errorf("parsing navigation history: %w", err)
	}

	if histResp.CurrentIndex >= len(histResp.Entries)-1 {
		return fmt.Errorf("no history to go forward to")
	}

	// Navigate to next entry
	nextEntry := histResp.Entries[histResp.CurrentIndex+1]
	_, err = c.CallSession(ctx, sessionID, "Page.navigateToHistoryEntry", map[string]interface{}{
		"entryId": nextEntry.ID,
	})
	if err != nil {
		return fmt.Errorf("navigating to history entry: %w", err)
	}

	return nil
}

// Reload reloads the page. If ignoreCache is true, the browser cache is bypassed.
func (c *Client) Reload(ctx context.Context, targetID string, ignoreCache bool) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Page domain on the session
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Page domain: %w", err)
	}

	// Reload
	params := map[string]interface{}{}
	if ignoreCache {
		params["ignoreCache"] = true
	}

	_, err = c.CallSession(ctx, sessionID, "Page.reload", params)
	if err != nil {
		return fmt.Errorf("reloading: %w", err)
	}

	return nil
}

// Screenshot captures a screenshot of a target.
func (c *Client) Screenshot(ctx context.Context, targetID string, opts ScreenshotOptions) ([]byte, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Build screenshot params
	params := map[string]interface{}{}
	if opts.Format != "" {
		params["format"] = opts.Format
	}
	if opts.Quality > 0 {
		params["quality"] = opts.Quality
	}

	// Capture screenshot
	result, err := c.CallSession(ctx, sessionID, "Page.captureScreenshot", params)
	if err != nil {
		return nil, fmt.Errorf("capturing screenshot: %w", err)
	}

	var screenshotResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &screenshotResp); err != nil {
		return nil, fmt.Errorf("parsing screenshot response: %w", err)
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(screenshotResp.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding screenshot data: %w", err)
	}

	return data, nil
}

// Eval evaluates a JavaScript expression in a target's page context.
func (c *Client) Eval(ctx context.Context, targetID string, expression string) (*EvalResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Runtime domain
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Evaluate expression
	evalResult, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    expression,
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluating expression: %w", err)
	}

	var evalResp struct {
		Result struct {
			Type  string      `json:"type"`
			Value interface{} `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(evalResult, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing eval response: %w", err)
	}

	if evalResp.ExceptionDetails != nil {
		return nil, fmt.Errorf("JS exception: %s", evalResp.ExceptionDetails.Text)
	}

	return &EvalResult{
		Value: evalResp.Result.Value,
		Type:  evalResp.Result.Type,
	}, nil
}

// Query finds the first DOM element matching a CSS selector.
func (c *Client) Query(ctx context.Context, targetID string, selector string) (*QueryResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return nil, fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return nil, fmt.Errorf("parsing document response: %w", err)
	}

	// Query selector
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return nil, fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return nil, fmt.Errorf("parsing query response: %w", err)
	}

	// If not found, return empty result
	if queryResp.NodeID == 0 {
		return &QueryResult{NodeID: 0}, nil
	}

	// Describe the node to get tag name and attributes
	descResult, err := c.CallSession(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("describing node: %w", err)
	}

	var descResp struct {
		Node struct {
			NodeName   string   `json:"nodeName"`
			Attributes []string `json:"attributes"`
		} `json:"node"`
	}
	if err := json.Unmarshal(descResult, &descResp); err != nil {
		return nil, fmt.Errorf("parsing describe response: %w", err)
	}

	// Parse attributes (CDP returns flat array: [name, value, name, value, ...])
	attrs := make(map[string]string)
	for i := 0; i+1 < len(descResp.Node.Attributes); i += 2 {
		attrs[descResp.Node.Attributes[i]] = descResp.Node.Attributes[i+1]
	}

	return &QueryResult{
		NodeID:     queryResp.NodeID,
		TagName:    descResp.Node.NodeName,
		Attributes: attrs,
	}, nil
}

// QueryShadow finds an element inside a shadow DOM.
// hostSelector is the CSS selector for the shadow host element.
// innerSelector is the CSS selector to query within the shadow root.
func (c *Client) QueryShadow(ctx context.Context, targetID string, hostSelector string, innerSelector string) (*QueryResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return nil, fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return nil, fmt.Errorf("parsing document response: %w", err)
	}

	// Find the shadow host element
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": hostSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("querying host selector: %w", err)
	}

	var hostResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &hostResp); err != nil {
		return nil, fmt.Errorf("parsing host query response: %w", err)
	}

	if hostResp.NodeID == 0 {
		return nil, fmt.Errorf("shadow host not found: %s", hostSelector)
	}

	// Describe the host node to get its shadow root
	descResult, err := c.CallSession(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"nodeId": hostResp.NodeID,
		"depth":  1,
		"pierce": true,
	})
	if err != nil {
		return nil, fmt.Errorf("describing host node: %w", err)
	}

	var descResp struct {
		Node struct {
			ShadowRoots []struct {
				NodeID   int    `json:"nodeId"`
				NodeType int    `json:"nodeType"`
				NodeName string `json:"nodeName"`
			} `json:"shadowRoots"`
		} `json:"node"`
	}
	if err := json.Unmarshal(descResult, &descResp); err != nil {
		return nil, fmt.Errorf("parsing describe response: %w", err)
	}

	if len(descResp.Node.ShadowRoots) == 0 {
		return nil, fmt.Errorf("no shadow root found on element: %s", hostSelector)
	}

	shadowRootID := descResp.Node.ShadowRoots[0].NodeID

	// Query within the shadow root
	innerResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   shadowRootID,
		"selector": innerSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("querying shadow selector: %w", err)
	}

	var innerResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(innerResult, &innerResp); err != nil {
		return nil, fmt.Errorf("parsing shadow query response: %w", err)
	}

	if innerResp.NodeID == 0 {
		return &QueryResult{NodeID: 0}, nil
	}

	// Describe the inner node to get tag name and attributes
	innerDescResult, err := c.CallSession(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"nodeId": innerResp.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("describing inner node: %w", err)
	}

	var innerDescResp struct {
		Node struct {
			NodeName   string   `json:"nodeName"`
			Attributes []string `json:"attributes"`
		} `json:"node"`
	}
	if err := json.Unmarshal(innerDescResult, &innerDescResp); err != nil {
		return nil, fmt.Errorf("parsing inner describe response: %w", err)
	}

	// Parse attributes
	attrs := make(map[string]string)
	for i := 0; i+1 < len(innerDescResp.Node.Attributes); i += 2 {
		attrs[innerDescResp.Node.Attributes[i]] = innerDescResp.Node.Attributes[i+1]
	}

	return &QueryResult{
		NodeID:     innerResp.NodeID,
		TagName:    innerDescResp.Node.NodeName,
		Attributes: attrs,
	}, nil
}

// Click clicks on the first element matching a CSS selector.
func (c *Client) Click(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query selector
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Get box model for element coordinates
	boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("getting box model: %w", err)
	}

	var boxResp struct {
		Model struct {
			Content []float64 `json:"content"` // [x1,y1, x2,y2, x3,y3, x4,y4]
		} `json:"model"`
	}
	if err := json.Unmarshal(boxResult, &boxResp); err != nil {
		return fmt.Errorf("parsing box model response: %w", err)
	}

	// Calculate center point from content quad
	content := boxResp.Model.Content
	if len(content) < 8 {
		return fmt.Errorf("invalid box model")
	}
	x := (content[0] + content[2] + content[4] + content[6]) / 4
	y := (content[1] + content[3] + content[5] + content[7]) / 4

	// Dispatch mouse events: move, press, release
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mousePressed: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseReleased: %w", err)
	}

	return nil
}

// ClickAt clicks at specific x, y coordinates.
func (c *Client) ClickAt(ctx context.Context, targetID string, x, y float64) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Dispatch mouse events: move, press, release
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mousePressed: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseReleased: %w", err)
	}

	return nil
}

// Tap performs a touch tap on an element (like a finger tap on mobile).
func (c *Client) Tap(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query selector
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Get element bounding box
	boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("getting box model: %w", err)
	}

	var boxResp struct {
		Model struct {
			Content []float64 `json:"content"`
		} `json:"model"`
	}
	if err := json.Unmarshal(boxResult, &boxResp); err != nil {
		return fmt.Errorf("parsing box model response: %w", err)
	}

	content := boxResp.Model.Content
	if len(content) < 8 {
		return fmt.Errorf("invalid box model")
	}
	x := (content[0] + content[2] + content[4] + content[6]) / 4
	y := (content[1] + content[3] + content[5] + content[7]) / 4

	// Dispatch touch events: touchStart, touchEnd
	touchPoints := []map[string]interface{}{
		{
			"x": x,
			"y": y,
		},
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type":        "touchStart",
		"touchPoints": touchPoints,
	})
	if err != nil {
		return fmt.Errorf("dispatching touchStart: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type":        "touchEnd",
		"touchPoints": []map[string]interface{}{},
	})
	if err != nil {
		return fmt.Errorf("dispatching touchEnd: %w", err)
	}

	return nil
}

// Drag performs a drag from one element to another.
func (c *Client) Drag(ctx context.Context, targetID string, sourceSelector, destSelector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Helper to get element center
	getCenter := func(selector string) (float64, float64, error) {
		queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
			"nodeId":   docResp.Root.NodeID,
			"selector": selector,
		})
		if err != nil {
			return 0, 0, fmt.Errorf("querying selector %s: %w", selector, err)
		}

		var queryResp struct {
			NodeID int `json:"nodeId"`
		}
		if err := json.Unmarshal(queryResult, &queryResp); err != nil {
			return 0, 0, fmt.Errorf("parsing query response: %w", err)
		}
		if queryResp.NodeID == 0 {
			return 0, 0, fmt.Errorf("element not found: %s", selector)
		}

		boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
			"nodeId": queryResp.NodeID,
		})
		if err != nil {
			return 0, 0, fmt.Errorf("getting box model: %w", err)
		}

		var boxResp struct {
			Model struct {
				Content []float64 `json:"content"`
			} `json:"model"`
		}
		if err := json.Unmarshal(boxResult, &boxResp); err != nil {
			return 0, 0, fmt.Errorf("parsing box model: %w", err)
		}

		content := boxResp.Model.Content
		if len(content) < 8 {
			return 0, 0, fmt.Errorf("invalid box model")
		}
		x := (content[0] + content[2] + content[4] + content[6]) / 4
		y := (content[1] + content[3] + content[5] + content[7]) / 4
		return x, y, nil
	}

	// Get source and destination centers
	srcX, srcY, err := getCenter(sourceSelector)
	if err != nil {
		return err
	}
	dstX, dstY, err := getCenter(destSelector)
	if err != nil {
		return err
	}

	// Perform drag: move to source, press, move to dest, release
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    srcX,
		"y":    srcY,
	})
	if err != nil {
		return fmt.Errorf("moving to source: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          srcX,
		"y":          srcY,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("pressing at source: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    dstX,
		"y":    dstY,
	})
	if err != nil {
		return fmt.Errorf("moving to destination: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          dstX,
		"y":          dstY,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("releasing at destination: %w", err)
	}

	return nil
}

// Fill fills an input element with text.
func (c *Client) Fill(ctx context.Context, targetID string, selector string, text string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query selector
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Focus the element
	_, err = c.CallSession(ctx, sessionID, "DOM.focus", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("focusing element: %w", err)
	}

	// Enable Runtime to clear value via JS
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Clear the input value using JavaScript
	_, err = c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf(`document.querySelector(%q).value = ''`, selector),
	})
	if err != nil {
		return fmt.Errorf("clearing input value: %w", err)
	}

	// Insert the text
	_, err = c.CallSession(ctx, sessionID, "Input.insertText", map[string]interface{}{
		"text": text,
	})
	if err != nil {
		return fmt.Errorf("inserting text: %w", err)
	}

	return nil
}

// GetHTML returns the outer HTML of an element matching the selector.
func (c *Client) GetHTML(ctx context.Context, targetID string, selector string) (string, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return "", err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return "", fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return "", fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return "", fmt.Errorf("parsing document response: %w", err)
	}

	// Query selector
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return "", fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return "", fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return "", fmt.Errorf("element not found: %s", selector)
	}

	// Get outer HTML
	htmlResult, err := c.CallSession(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return "", fmt.Errorf("getting outer HTML: %w", err)
	}

	var htmlResp struct {
		OuterHTML string `json:"outerHTML"`
	}
	if err := json.Unmarshal(htmlResult, &htmlResp); err != nil {
		return "", fmt.Errorf("parsing HTML response: %w", err)
	}

	return htmlResp.OuterHTML, nil
}

// WaitFor waits for an element matching the selector to appear.
func (c *Client) WaitFor(ctx context.Context, targetID string, selector string, timeout time.Duration) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for selector: %s", selector)
		}

		// Get document root
		docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
		if err != nil {
			return fmt.Errorf("getting document: %w", err)
		}

		var docResp struct {
			Root struct {
				NodeID int `json:"nodeId"`
			} `json:"root"`
		}
		if err := json.Unmarshal(docResult, &docResp); err != nil {
			return fmt.Errorf("parsing document response: %w", err)
		}

		// Query selector
		queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
			"nodeId":   docResp.Root.NodeID,
			"selector": selector,
		})
		if err != nil {
			return fmt.Errorf("querying selector: %w", err)
		}

		var queryResp struct {
			NodeID int `json:"nodeId"`
		}
		if err := json.Unmarshal(queryResult, &queryResp); err != nil {
			return fmt.Errorf("parsing query response: %w", err)
		}

		// Found!
		if queryResp.NodeID != 0 {
			return nil
		}

		// Wait before polling again
		time.Sleep(pollInterval)
	}
}

// GetText returns the text content of an element.
func (c *Client) GetText(ctx context.Context, targetID string, selector string) (string, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return "", err
	}

	// Enable Runtime domain to use JavaScript
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return "", fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Use JavaScript to get innerText (handles whitespace better than textContent)
	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    fmt.Sprintf(`document.querySelector(%q)?.innerText || ''`, selector),
		"returnByValue": true,
	})
	if err != nil {
		return "", fmt.Errorf("evaluating expression: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return "", fmt.Errorf("parsing eval response: %w", err)
	}

	return evalResp.Result.Value, nil
}

// Type sends individual key events for each character in the text.
// This is useful for inputs that need realistic typing (autocomplete, etc.).
func (c *Client) Type(ctx context.Context, targetID string, text string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Input domain
	_, err = c.CallSession(ctx, sessionID, "Input.enable", nil)
	if err != nil {
		// Input.enable might not exist in all Chrome versions, continue anyway
	}

	// Type each character individually
	for _, char := range text {
		charStr := string(char)

		// keyDown with char
		_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
			"type": "keyDown",
			"text": charStr,
			"key":  charStr,
		})
		if err != nil {
			return fmt.Errorf("keyDown for %q: %w", charStr, err)
		}

		// keyUp
		_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
			"type": "keyUp",
			"key":  charStr,
		})
		if err != nil {
			return fmt.Errorf("keyUp for %q: %w", charStr, err)
		}
	}

	return nil
}

// keyCodeMap maps key names to their key codes.
var keyCodeMap = map[string]int{
	"Enter":      13,
	"Tab":        9,
	"Escape":     27,
	"Backspace":  8,
	"Delete":     46,
	"ArrowUp":    38,
	"ArrowDown":  40,
	"ArrowLeft":  37,
	"ArrowRight": 39,
	"Home":       36,
	"End":        35,
	"PageUp":     33,
	"PageDown":   34,
	"Space":      32,
}

// KeyModifiers represents keyboard modifier keys.
type KeyModifiers struct {
	Ctrl  bool
	Alt   bool
	Shift bool
	Meta  bool
}

// modifierBitmask returns the CDP modifier bitmask.
func (m KeyModifiers) modifierBitmask() int {
	mask := 0
	if m.Shift {
		mask |= 1
	}
	if m.Ctrl {
		mask |= 2
	}
	if m.Alt {
		mask |= 4
	}
	if m.Meta {
		mask |= 8
	}
	return mask
}

// PressKey presses a special key (Enter, Tab, Escape, etc.).
func (c *Client) PressKey(ctx context.Context, targetID string, key string) error {
	return c.PressKeyWithModifiers(ctx, targetID, key, KeyModifiers{})
}

// PressKeyWithModifiers presses a key with modifier keys (Ctrl, Alt, Shift, Meta).
func (c *Client) PressKeyWithModifiers(ctx context.Context, targetID string, key string, mods KeyModifiers) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Get key code if available
	keyCode, hasKeyCode := keyCodeMap[key]
	modMask := mods.modifierBitmask()

	params := map[string]interface{}{
		"type":      "keyDown",
		"key":       key,
		"modifiers": modMask,
	}
	if hasKeyCode {
		params["windowsVirtualKeyCode"] = keyCode
		params["nativeVirtualKeyCode"] = keyCode
	}

	// keyDown
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", params)
	if err != nil {
		return fmt.Errorf("keyDown for %q: %w", key, err)
	}

	// keyUp
	params["type"] = "keyUp"
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", params)
	if err != nil {
		return fmt.Errorf("keyUp for %q: %w", key, err)
	}

	return nil
}

// CaptureConsole starts capturing console messages from a page.
// Returns a channel that receives ConsoleMessage and a stop function.
// The stop function MUST be called when done to release resources.
// The channel is closed when stop is called or when the client is closed.
func (c *Client) CaptureConsole(ctx context.Context, targetID string) (<-chan ConsoleMessage, func(), error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, nil, err
	}

	// Enable Runtime domain to receive console events
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Subscribe to console API events
	eventCh := c.subscribeEvent(sessionID, "Runtime.consoleAPICalled")

	// Create output channel
	output := make(chan ConsoleMessage, 100)

	// Create a done channel to signal the goroutine to stop
	done := make(chan struct{})
	var stopOnce sync.Once

	// Stop function to clean up resources
	stop := func() {
		stopOnce.Do(func() {
			close(done)
			c.unsubscribeEvent(sessionID, "Runtime.consoleAPICalled", eventCh)
			// Best effort to disable Runtime domain
			disableCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			c.CallSession(disableCtx, sessionID, "Runtime.disable", nil)
		})
	}

	// Start goroutine to translate events to ConsoleMessages
	go func() {
		defer close(output)
		for {
			select {
			case params, ok := <-eventCh:
				if !ok {
					return
				}
				// Parse the event
				var event struct {
					Type string `json:"type"`
					Args []struct {
						Type  string      `json:"type"`
						Value interface{} `json:"value"`
					} `json:"args"`
				}
				if err := json.Unmarshal(params, &event); err != nil {
					continue
				}

				// Build message text from args
				var text string
				for i, arg := range event.Args {
					if i > 0 {
						text += " "
					}
					if arg.Value != nil {
						text += fmt.Sprintf("%v", arg.Value)
					}
				}

				select {
				case output <- ConsoleMessage{Type: event.Type, Text: text}:
				default:
					// Drop if channel is full
				}
			case <-done:
				return
			case <-c.closeCh:
				return
			}
		}
	}()

	return output, stop, nil
}

// ExceptionInfo represents a JavaScript exception.
type ExceptionInfo struct {
	Text        string `json:"text"`
	LineNumber  int    `json:"lineNumber,omitempty"`
	ColumnNumber int   `json:"columnNumber,omitempty"`
	URL         string `json:"url,omitempty"`
}

// CaptureExceptions starts capturing JavaScript exceptions from a page.
// Returns a channel that receives ExceptionInfo and a stop function.
// The stop function MUST be called when done to release resources.
// The channel is closed when stop is called or when the client is closed.
func (c *Client) CaptureExceptions(ctx context.Context, targetID string) (<-chan ExceptionInfo, func(), error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, nil, err
	}

	// Enable Runtime domain to receive exception events
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Subscribe to exception events
	eventCh := c.subscribeEvent(sessionID, "Runtime.exceptionThrown")

	// Create output channel
	output := make(chan ExceptionInfo, 100)

	// Create a done channel to signal the goroutine to stop
	done := make(chan struct{})
	var stopOnce sync.Once

	// Stop function to clean up resources
	stop := func() {
		stopOnce.Do(func() {
			close(done)
			c.unsubscribeEvent(sessionID, "Runtime.exceptionThrown", eventCh)
			// Best effort to disable Runtime domain
			disableCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			c.CallSession(disableCtx, sessionID, "Runtime.disable", nil)
		})
	}

	// Start goroutine to translate events to ExceptionInfo
	go func() {
		defer close(output)
		for {
			select {
			case params, ok := <-eventCh:
				if !ok {
					return
				}
				// Parse the event
				var event struct {
					ExceptionDetails struct {
						Text         string `json:"text"`
						LineNumber   int    `json:"lineNumber"`
						ColumnNumber int    `json:"columnNumber"`
						URL          string `json:"url"`
						Exception    struct {
							Description string `json:"description"`
						} `json:"exception"`
					} `json:"exceptionDetails"`
				}
				if err := json.Unmarshal(params, &event); err != nil {
					continue
				}

				text := event.ExceptionDetails.Text
				if event.ExceptionDetails.Exception.Description != "" {
					text = event.ExceptionDetails.Exception.Description
				}

				select {
				case output <- ExceptionInfo{
					Text:         text,
					LineNumber:   event.ExceptionDetails.LineNumber,
					ColumnNumber: event.ExceptionDetails.ColumnNumber,
					URL:          event.ExceptionDetails.URL,
				}:
				default:
					// Drop if channel is full
				}
			case <-done:
				return
			case <-c.closeCh:
				return
			}
		}
	}()

	return output, stop, nil
}

// CaptureNetwork starts capturing network events from a page.
// Returns a channel that receives NetworkEvent and a stop function.
// The stop function MUST be called when done to release resources.
// The channel is closed when stop is called or when the client is closed.
func (c *Client) CaptureNetwork(ctx context.Context, targetID string) (<-chan NetworkEvent, func(), error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, nil, err
	}

	// Enable Network domain to receive events
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("enabling Network domain: %w", err)
	}

	// Subscribe to network events
	requestCh := c.subscribeEvent(sessionID, "Network.requestWillBeSent")
	responseCh := c.subscribeEvent(sessionID, "Network.responseReceived")

	// Create output channel
	output := make(chan NetworkEvent, 100)

	// Create a done channel to signal the goroutine to stop
	done := make(chan struct{})
	var stopOnce sync.Once

	// Stop function to clean up resources
	stop := func() {
		stopOnce.Do(func() {
			close(done)
			c.unsubscribeEvent(sessionID, "Network.requestWillBeSent", requestCh)
			c.unsubscribeEvent(sessionID, "Network.responseReceived", responseCh)
			// Best effort to disable Network domain
			disableCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			c.CallSession(disableCtx, sessionID, "Network.disable", nil)
		})
	}

	// Start goroutine to translate events
	go func() {
		defer close(output)
		for {
			select {
			case params, ok := <-requestCh:
				if !ok {
					return
				}
				var event struct {
					RequestID string `json:"requestId"`
					Request   struct {
						URL    string `json:"url"`
						Method string `json:"method"`
					} `json:"request"`
				}
				if err := json.Unmarshal(params, &event); err != nil {
					continue
				}
				select {
				case output <- NetworkEvent{
					Type:      "request",
					RequestID: event.RequestID,
					URL:       event.Request.URL,
					Method:    event.Request.Method,
				}:
				default:
				}
			case params, ok := <-responseCh:
				if !ok {
					return
				}
				var event struct {
					RequestID string `json:"requestId"`
					Response  struct {
						URL      string `json:"url"`
						Status   int    `json:"status"`
						MimeType string `json:"mimeType"`
					} `json:"response"`
				}
				if err := json.Unmarshal(params, &event); err != nil {
					continue
				}
				select {
				case output <- NetworkEvent{
					Type:      "response",
					RequestID: event.RequestID,
					URL:       event.Response.URL,
					Status:    event.Response.Status,
					MimeType:  event.Response.MimeType,
				}:
				default:
				}
			case <-done:
				return
			case <-c.closeCh:
				return
			}
		}
	}()

	return output, stop, nil
}

// HARLog represents an HTTP Archive log.
type HARLog struct {
	Log struct {
		Version string     `json:"version"`
		Creator HARCreator `json:"creator"`
		Entries []HAREntry `json:"entries"`
	} `json:"log"`
}

// HARCreator represents the creator of the HAR file.
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HAREntry represents a single HTTP transaction.
type HAREntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Cache           struct{}    `json:"cache"`
	Timings         HARTimings  `json:"timings"`
}

// HARRequest represents an HTTP request.
type HARRequest struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
	QueryString []HARQuery  `json:"queryString"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

// HARResponse represents an HTTP response.
type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
	Content     HARContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

// HARHeader represents an HTTP header.
type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARQuery represents a query string parameter.
type HARQuery struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARContent represents the response body content.
type HARContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
}

// HARTimings represents the timing information.
type HARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// CaptureHAR captures network activity and returns it as a HAR log.
func (c *Client) CaptureHAR(ctx context.Context, targetID string, duration time.Duration) (*HARLog, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Network domain
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Network domain: %w", err)
	}

	// Subscribe to network events
	requestCh := c.subscribeEvent(sessionID, "Network.requestWillBeSent")
	responseCh := c.subscribeEvent(sessionID, "Network.responseReceived")
	loadingFinishedCh := c.subscribeEvent(sessionID, "Network.loadingFinished")

	defer func() {
		c.unsubscribeEvent(sessionID, "Network.requestWillBeSent", requestCh)
		c.unsubscribeEvent(sessionID, "Network.responseReceived", responseCh)
		c.unsubscribeEvent(sessionID, "Network.loadingFinished", loadingFinishedCh)
		disableCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		c.CallSession(disableCtx, sessionID, "Network.disable", nil)
	}()

	// Track requests and responses
	type requestInfo struct {
		startTime time.Time
		method    string
		url       string
		headers   map[string]string
	}
	type responseInfo struct {
		status   int
		mimeType string
		headers  map[string]string
	}
	requests := make(map[string]*requestInfo)
	responses := make(map[string]*responseInfo)
	timings := make(map[string]float64) // requestID -> duration in ms

	timeout := time.After(duration)

	for {
		select {
		case params, ok := <-requestCh:
			if !ok {
				goto done
			}
			var event struct {
				RequestID string `json:"requestId"`
				Timestamp float64 `json:"timestamp"`
				Request   struct {
					URL     string            `json:"url"`
					Method  string            `json:"method"`
					Headers map[string]string `json:"headers"`
				} `json:"request"`
			}
			if err := json.Unmarshal(params, &event); err != nil {
				continue
			}
			requests[event.RequestID] = &requestInfo{
				startTime: time.Now(),
				method:    event.Request.Method,
				url:       event.Request.URL,
				headers:   event.Request.Headers,
			}

		case params, ok := <-responseCh:
			if !ok {
				goto done
			}
			var event struct {
				RequestID string `json:"requestId"`
				Response  struct {
					Status   int               `json:"status"`
					MimeType string            `json:"mimeType"`
					Headers  map[string]string `json:"headers"`
				} `json:"response"`
			}
			if err := json.Unmarshal(params, &event); err != nil {
				continue
			}
			responses[event.RequestID] = &responseInfo{
				status:   event.Response.Status,
				mimeType: event.Response.MimeType,
				headers:  event.Response.Headers,
			}

		case params, ok := <-loadingFinishedCh:
			if !ok {
				goto done
			}
			var event struct {
				RequestID string  `json:"requestId"`
				Timestamp float64 `json:"timestamp"`
			}
			if err := json.Unmarshal(params, &event); err != nil {
				continue
			}
			if req, ok := requests[event.RequestID]; ok {
				timings[event.RequestID] = float64(time.Since(req.startTime).Milliseconds())
			}

		case <-timeout:
			goto done

		case <-ctx.Done():
			return nil, ctx.Err()

		case <-c.closeCh:
			goto done
		}
	}

done:
	// Build HAR log
	har := &HARLog{}
	har.Log.Version = "1.2"
	har.Log.Creator = HARCreator{Name: "cdp-cli", Version: "1.0"}
	har.Log.Entries = make([]HAREntry, 0)

	for requestID, req := range requests {
		entry := HAREntry{
			StartedDateTime: req.startTime.Format(time.RFC3339Nano),
			Time:            timings[requestID],
			Request: HARRequest{
				Method:      req.method,
				URL:         req.url,
				HTTPVersion: "HTTP/1.1",
				Headers:     make([]HARHeader, 0),
				QueryString: make([]HARQuery, 0),
				HeadersSize: -1,
				BodySize:    -1,
			},
			Response: HARResponse{
				Status:      0,
				StatusText:  "",
				HTTPVersion: "HTTP/1.1",
				Headers:     make([]HARHeader, 0),
				Content:     HARContent{Size: -1, MimeType: ""},
				RedirectURL: "",
				HeadersSize: -1,
				BodySize:    -1,
			},
			Timings: HARTimings{
				Send:    -1,
				Wait:    -1,
				Receive: -1,
			},
		}

		// Add request headers
		for name, value := range req.headers {
			entry.Request.Headers = append(entry.Request.Headers, HARHeader{Name: name, Value: value})
		}

		// Add response if available
		if resp, ok := responses[requestID]; ok {
			entry.Response.Status = resp.status
			entry.Response.Content.MimeType = resp.mimeType
			for name, value := range resp.headers {
				entry.Response.Headers = append(entry.Response.Headers, HARHeader{Name: name, Value: value})
			}
		}

		har.Log.Entries = append(har.Log.Entries, entry)
	}

	return har, nil
}

// CoverageResult represents JavaScript code coverage data.
type CoverageResult struct {
	Scripts []ScriptCoverage `json:"scripts"`
}

// ScriptCoverage represents coverage for a single script.
type ScriptCoverage struct {
	ScriptID string          `json:"scriptId"`
	URL      string          `json:"url"`
	Ranges   []CoverageRange `json:"ranges"`
}

// CoverageRange represents a covered range in the script.
type CoverageRange struct {
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
	Count       int `json:"count"`
}

// GetCoverage returns JavaScript code coverage data.
func (c *Client) GetCoverage(ctx context.Context, targetID string) (*CoverageResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Profiler domain
	_, err = c.CallSession(ctx, sessionID, "Profiler.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Profiler domain: %w", err)
	}

	// Start precise coverage
	_, err = c.CallSession(ctx, sessionID, "Profiler.startPreciseCoverage", map[string]interface{}{
		"callCount": true,
		"detailed":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("starting coverage: %w", err)
	}

	// Get coverage data
	result, err := c.CallSession(ctx, sessionID, "Profiler.takePreciseCoverage", nil)
	if err != nil {
		return nil, fmt.Errorf("taking coverage: %w", err)
	}

	// Stop coverage collection
	c.CallSession(ctx, sessionID, "Profiler.stopPreciseCoverage", nil)
	c.CallSession(ctx, sessionID, "Profiler.disable", nil)

	var resp struct {
		Result []struct {
			ScriptID  string `json:"scriptId"`
			URL       string `json:"url"`
			Functions []struct {
				FunctionName string `json:"functionName"`
				Ranges       []struct {
					StartOffset int `json:"startOffset"`
					EndOffset   int `json:"endOffset"`
					Count       int `json:"count"`
				} `json:"ranges"`
			} `json:"functions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing coverage response: %w", err)
	}

	coverage := &CoverageResult{
		Scripts: make([]ScriptCoverage, 0, len(resp.Result)),
	}

	for _, script := range resp.Result {
		sc := ScriptCoverage{
			ScriptID: script.ScriptID,
			URL:      script.URL,
			Ranges:   make([]CoverageRange, 0),
		}

		// Flatten function ranges into script ranges
		for _, fn := range script.Functions {
			for _, r := range fn.Ranges {
				sc.Ranges = append(sc.Ranges, CoverageRange{
					StartOffset: r.StartOffset,
					EndOffset:   r.EndOffset,
					Count:       r.Count,
				})
			}
		}

		coverage.Scripts = append(coverage.Scripts, sc)
	}

	return coverage, nil
}

// StylesheetInfo represents information about a stylesheet.
type StylesheetInfo struct {
	StyleSheetID string `json:"styleSheetId"`
	SourceURL    string `json:"sourceURL"`
	Title        string `json:"title"`
	Disabled     bool   `json:"disabled"`
	IsInline     bool   `json:"isInline"`
	Length       int    `json:"length"`
}

// StylesheetsResult represents all stylesheets on the page.
type StylesheetsResult struct {
	Stylesheets []StylesheetInfo `json:"stylesheets"`
}

// GetStylesheets returns all stylesheets on the page.
func (c *Client) GetStylesheets(ctx context.Context, targetID string) (*StylesheetsResult, error) {
	// Use JavaScript to get stylesheet information from document.styleSheets
	result, err := c.Eval(ctx, targetID, `
		(function() {
			const sheets = [];
			for (let i = 0; i < document.styleSheets.length; i++) {
				const sheet = document.styleSheets[i];
				let cssText = '';
				let ruleCount = 0;
				try {
					if (sheet.cssRules) {
						ruleCount = sheet.cssRules.length;
					}
				} catch (e) {
					// CORS restrictions may prevent access to cssRules
				}
				sheets.push({
					styleSheetId: i.toString(),
					sourceURL: sheet.href || '',
					title: sheet.title || '',
					disabled: sheet.disabled,
					isInline: !sheet.href,
					length: ruleCount
				});
			}
			return sheets;
		})()
	`)
	if err != nil {
		return nil, fmt.Errorf("getting stylesheets: %w", err)
	}

	stylesheets := &StylesheetsResult{
		Stylesheets: make([]StylesheetInfo, 0),
	}

	if result.Value == nil {
		return stylesheets, nil
	}

	// Parse the result array
	sheetsData, ok := result.Value.([]interface{})
	if !ok {
		return stylesheets, nil
	}

	for _, item := range sheetsData {
		sheet, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		info := StylesheetInfo{}
		if id, ok := sheet["styleSheetId"].(string); ok {
			info.StyleSheetID = id
		}
		if url, ok := sheet["sourceURL"].(string); ok {
			info.SourceURL = url
		}
		if title, ok := sheet["title"].(string); ok {
			info.Title = title
		}
		if disabled, ok := sheet["disabled"].(bool); ok {
			info.Disabled = disabled
		}
		if isInline, ok := sheet["isInline"].(bool); ok {
			info.IsInline = isInline
		}
		if length, ok := sheet["length"].(float64); ok {
			info.Length = int(length)
		}

		stylesheets.Stylesheets = append(stylesheets.Stylesheets, info)
	}

	return stylesheets, nil
}

// PageInfo represents combined information about the current page.
type PageInfo struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	ReadyState  string `json:"readyState"`
	CharacterSet string `json:"characterSet"`
	ContentType string `json:"contentType"`
}

// GetPageInfo returns combined information about the current page.
func (c *Client) GetPageInfo(ctx context.Context, targetID string) (*PageInfo, error) {
	result, err := c.Eval(ctx, targetID, `
		(function() {
			return {
				title: document.title,
				url: document.location.href,
				readyState: document.readyState,
				characterSet: document.characterSet,
				contentType: document.contentType
			};
		})()
	`)
	if err != nil {
		return nil, fmt.Errorf("getting page info: %w", err)
	}

	info := &PageInfo{}
	if result.Value == nil {
		return info, nil
	}

	data, ok := result.Value.(map[string]interface{})
	if !ok {
		return info, nil
	}

	if title, ok := data["title"].(string); ok {
		info.Title = title
	}
	if url, ok := data["url"].(string); ok {
		info.URL = url
	}
	if readyState, ok := data["readyState"].(string); ok {
		info.ReadyState = readyState
	}
	if characterSet, ok := data["characterSet"].(string); ok {
		info.CharacterSet = characterSet
	}
	if contentType, ok := data["contentType"].(string); ok {
		info.ContentType = contentType
	}

	return info, nil
}

// WaitForText waits for text to appear on the page.
func (c *Client) WaitForText(ctx context.Context, targetID string, text string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	escapedText := strings.ReplaceAll(text, "'", "\\'")

	for time.Now().Before(deadline) {
		result, err := c.Eval(ctx, targetID, fmt.Sprintf(
			"document.body && document.body.innerText.includes('%s')", escapedText))
		if err != nil {
			return err
		}

		if result.Value == true {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return fmt.Errorf("timeout waiting for text %q", text)
}

// ScriptInfo represents information about a script element.
type ScriptInfo struct {
	Src    string `json:"src"`
	Type   string `json:"type"`
	Async  bool   `json:"async"`
	Defer  bool   `json:"defer"`
	Inline bool   `json:"inline"`
}

// ScriptsResult represents all scripts on the page.
type ScriptsResult struct {
	Scripts []ScriptInfo `json:"scripts"`
}

// GetScripts returns all script elements on the page.
func (c *Client) GetScripts(ctx context.Context, targetID string) (*ScriptsResult, error) {
	result, err := c.Eval(ctx, targetID, `
		(function() {
			const scripts = [];
			const elements = document.querySelectorAll('script');
			for (const el of elements) {
				scripts.push({
					src: el.src || '',
					type: el.type || '',
					async: el.async,
					defer: el.defer,
					inline: !el.src
				});
			}
			return scripts;
		})()
	`)
	if err != nil {
		return nil, fmt.Errorf("getting scripts: %w", err)
	}

	scripts := &ScriptsResult{
		Scripts: make([]ScriptInfo, 0),
	}

	if result.Value == nil {
		return scripts, nil
	}

	scriptsData, ok := result.Value.([]interface{})
	if !ok {
		return scripts, nil
	}

	for _, item := range scriptsData {
		script, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		info := ScriptInfo{}
		if src, ok := script["src"].(string); ok {
			info.Src = src
		}
		if typ, ok := script["type"].(string); ok {
			info.Type = typ
		}
		if async, ok := script["async"].(bool); ok {
			info.Async = async
		}
		if deferred, ok := script["defer"].(bool); ok {
			info.Defer = deferred
		}
		if inline, ok := script["inline"].(bool); ok {
			info.Inline = inline
		}

		scripts.Scripts = append(scripts.Scripts, info)
	}

	return scripts, nil
}

// FindResult represents the result of finding text on the page.
type FindResult struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
	Found bool   `json:"found"`
}

// FindText searches for text on the page and returns occurrence count.
func (c *Client) FindText(ctx context.Context, targetID string, text string) (*FindResult, error) {
	escapedText := strings.ReplaceAll(text, "'", "\\'")
	result, err := c.Eval(ctx, targetID, fmt.Sprintf(`
		(function() {
			const text = '%s';
			const content = document.body ? document.body.innerText : '';
			let count = 0;
			let pos = 0;
			while ((pos = content.indexOf(text, pos)) !== -1) {
				count++;
				pos += text.length;
			}
			return { text: text, count: count, found: count > 0 };
		})()
	`, escapedText))
	if err != nil {
		return nil, fmt.Errorf("finding text: %w", err)
	}

	findResult := &FindResult{Text: text}
	if result.Value == nil {
		return findResult, nil
	}

	data, ok := result.Value.(map[string]interface{})
	if !ok {
		return findResult, nil
	}

	if count, ok := data["count"].(float64); ok {
		findResult.Count = int(count)
	}
	if found, ok := data["found"].(bool); ok {
		findResult.Found = found
	}

	return findResult, nil
}

// SetValueResult represents the result of setting an input value.
type SetValueResult struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

// SetValue directly sets the value of an input/textarea element.
func (c *Client) SetValue(ctx context.Context, targetID string, selector string, value string) (*SetValueResult, error) {
	escapedSelector := strings.ReplaceAll(selector, "'", "\\'")
	escapedValue := strings.ReplaceAll(value, "'", "\\'")
	escapedValue = strings.ReplaceAll(escapedValue, "\n", "\\n")

	result, err := c.Eval(ctx, targetID, fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) {
				return { error: 'Element not found: %s' };
			}
			el.value = '%s';
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return { selector: '%s', value: el.value };
		})()
	`, escapedSelector, escapedSelector, escapedValue, escapedSelector))
	if err != nil {
		return nil, fmt.Errorf("setting value: %w", err)
	}

	setResult := &SetValueResult{Selector: selector, Value: value}
	if result.Value != nil {
		if data, ok := result.Value.(map[string]interface{}); ok {
			if errMsg, ok := data["error"].(string); ok {
				return nil, fmt.Errorf("%s", errMsg)
			}
			if v, ok := data["value"].(string); ok {
				setResult.Value = v
			}
		}
	}

	return setResult, nil
}

// MouseMoveResult represents the result of moving the mouse.
type MouseMoveResult struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// MouseMove moves the mouse to the specified coordinates without clicking.
func (c *Client) MouseMove(ctx context.Context, targetID string, x, y float64) (*MouseMoveResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return nil, fmt.Errorf("moving mouse: %w", err)
	}

	return &MouseMoveResult{X: x, Y: y}, nil
}

// GetCookies returns all cookies for the page.
func (c *Client) GetCookies(ctx context.Context, targetID string) ([]Cookie, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Get cookies via Network domain
	result, err := c.CallSession(ctx, sessionID, "Network.getCookies", nil)
	if err != nil {
		return nil, fmt.Errorf("getting cookies: %w", err)
	}

	var resp struct {
		Cookies []Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing cookies response: %w", err)
	}

	return resp.Cookies, nil
}

// SetCookie sets a cookie for the page.
func (c *Client) SetCookie(ctx context.Context, targetID string, cookie Cookie) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"name":  cookie.Name,
		"value": cookie.Value,
	}
	if cookie.Domain != "" {
		params["domain"] = cookie.Domain
	}
	if cookie.Path != "" {
		params["path"] = cookie.Path
	}
	if cookie.Expires > 0 {
		params["expires"] = cookie.Expires
	}
	if cookie.HTTPOnly {
		params["httpOnly"] = cookie.HTTPOnly
	}
	if cookie.Secure {
		params["secure"] = cookie.Secure
	}
	if cookie.SameSite != "" {
		params["sameSite"] = cookie.SameSite
	}

	_, err = c.CallSession(ctx, sessionID, "Network.setCookie", params)
	if err != nil {
		return fmt.Errorf("setting cookie: %w", err)
	}

	return nil
}

// DeleteCookie deletes a cookie by name and domain.
func (c *Client) DeleteCookie(ctx context.Context, targetID string, name, domain string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"name": name,
	}
	if domain != "" {
		params["domain"] = domain
	}

	_, err = c.CallSession(ctx, sessionID, "Network.deleteCookies", params)
	if err != nil {
		return fmt.Errorf("deleting cookie: %w", err)
	}

	return nil
}

// ClearCookies clears all cookies.
func (c *Client) ClearCookies(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Network.clearBrowserCookies", nil)
	if err != nil {
		return fmt.Errorf("clearing cookies: %w", err)
	}

	return nil
}

// Focus focuses on an element specified by selector.
func (c *Client) Focus(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM and Runtime domains
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query for element
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Focus on the element
	_, err = c.CallSession(ctx, sessionID, "DOM.focus", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("focusing element: %w", err)
	}

	return nil
}

// NewTab creates a new browser tab and returns its target ID.
func (c *Client) NewTab(ctx context.Context, url string) (string, error) {
	if url == "" {
		url = "about:blank"
	}

	result, err := c.Call(ctx, "Target.createTarget", map[string]interface{}{
		"url": url,
	})
	if err != nil {
		return "", fmt.Errorf("creating target: %w", err)
	}

	var resp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	return resp.TargetID, nil
}

// ScrollIntoView scrolls an element into view.
func (c *Client) ScrollIntoView(ctx context.Context, targetID string, selector string) error {
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) throw new Error('Element not found');
			el.scrollIntoView({ behavior: 'instant', block: 'center' });
			return true;
		})()
	`, selector)

	_, err := c.Eval(ctx, targetID, js)
	return err
}

// ScrollBy scrolls the page by x and y pixels.
func (c *Client) ScrollBy(ctx context.Context, targetID string, x, y int) error {
	js := fmt.Sprintf(`window.scrollBy(%d, %d); true`, x, y)
	_, err := c.Eval(ctx, targetID, js)
	return err
}

// CountElements returns the number of elements matching the selector.
func (c *Client) CountElements(ctx context.Context, targetID string, selector string) (int, error) {
	js := fmt.Sprintf(`document.querySelectorAll(%q).length`, selector)
	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return 0, err
	}
	if result.Value == nil {
		return 0, nil
	}
	// JSON numbers are float64
	if f, ok := result.Value.(float64); ok {
		return int(f), nil
	}
	return 0, fmt.Errorf("unexpected type: %T", result.Value)
}

// IsVisible checks if an element is visible.
func (c *Client) IsVisible(ctx context.Context, targetID string, selector string) (bool, error) {
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) return false;
			const style = window.getComputedStyle(el);
			const rect = el.getBoundingClientRect();
			return style.display !== 'none' &&
			       style.visibility !== 'hidden' &&
			       style.opacity !== '0' &&
			       rect.width > 0 && rect.height > 0;
		})()
	`, selector)

	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return false, err
	}
	if b, ok := result.Value.(bool); ok {
		return b, nil
	}
	return false, nil
}

// BoundingBox represents an element's position and size.
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// DeviceInfo contains device emulation parameters.
type DeviceInfo struct {
	Name              string  `json:"name"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
	Mobile            bool    `json:"mobile"`
	UserAgent         string  `json:"userAgent"`
}

// CommonDevices is a map of common device names to their configurations.
var CommonDevices = map[string]DeviceInfo{
	"iPhone 12": {
		Name:              "iPhone 12",
		Width:             390,
		Height:            844,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPhone 12 Pro": {
		Name:              "iPhone 12 Pro",
		Width:             390,
		Height:            844,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPhone 12 Pro Max": {
		Name:              "iPhone 12 Pro Max",
		Width:             428,
		Height:            926,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPhone SE": {
		Name:              "iPhone SE",
		Width:             375,
		Height:            667,
		DeviceScaleFactor: 2,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"Pixel 5": {
		Name:              "Pixel 5",
		Width:             393,
		Height:            851,
		DeviceScaleFactor: 2.75,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36",
	},
	"Galaxy S21": {
		Name:              "Galaxy S21",
		Width:             360,
		Height:            800,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36",
	},
	"iPad": {
		Name:              "iPad",
		Width:             768,
		Height:            1024,
		DeviceScaleFactor: 2,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPad Pro": {
		Name:              "iPad Pro",
		Width:             1024,
		Height:            1366,
		DeviceScaleFactor: 2,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
}

// ElementLayout contains comprehensive layout information for an element.
type ElementLayout struct {
	Selector string                 `json:"selector"`
	TagName  string                 `json:"tagName"`
	Bounds   *BoundingBox           `json:"bounds"`
	Styles   map[string]string      `json:"styles,omitempty"`
	Children []ElementLayout        `json:"children,omitempty"`
}

// GetComputedStyles returns computed CSS styles for an element.
func (c *Client) GetComputedStyles(ctx context.Context, targetID string, selector string, properties []string) (map[string]string, error) {
	// Build JS to get computed styles
	propsJS := "null"
	if len(properties) > 0 {
		propsJSON, _ := json.Marshal(properties)
		propsJS = string(propsJSON)
	}

	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) return null;
			const computed = window.getComputedStyle(el);
			const props = %s;
			const result = {};
			if (props) {
				for (const p of props) {
					result[p] = computed.getPropertyValue(p);
				}
			} else {
				// Return common layout/styling properties
				const common = [
					'display', 'position', 'top', 'left', 'right', 'bottom',
					'width', 'height', 'minWidth', 'minHeight', 'maxWidth', 'maxHeight',
					'margin', 'marginTop', 'marginRight', 'marginBottom', 'marginLeft',
					'padding', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft',
					'border', 'borderWidth', 'borderStyle', 'borderColor',
					'backgroundColor', 'color', 'fontSize', 'fontFamily', 'fontWeight',
					'lineHeight', 'textAlign', 'overflow', 'visibility', 'opacity',
					'zIndex', 'flexDirection', 'justifyContent', 'alignItems',
					'gridTemplateColumns', 'gridTemplateRows', 'gap'
				];
				for (const p of common) {
					const val = computed.getPropertyValue(p.replace(/([A-Z])/g, '-$1').toLowerCase());
					if (val) result[p] = val;
				}
			}
			return result;
		})()
	`, selector, propsJS)

	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return nil, err
	}

	if result.Value == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	stylesMap, ok := result.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	styles := make(map[string]string)
	for k, v := range stylesMap {
		if s, ok := v.(string); ok {
			styles[k] = s
		}
	}

	return styles, nil
}

// GetElementLayout returns comprehensive layout info for an element and its children.
func (c *Client) GetElementLayout(ctx context.Context, targetID string, selector string, depth int) (*ElementLayout, error) {
	js := fmt.Sprintf(`
		(function() {
			function getLayout(el, currentDepth, maxDepth) {
				if (!el) return null;
				const rect = el.getBoundingClientRect();
				const computed = window.getComputedStyle(el);

				const layout = {
					tagName: el.tagName,
					bounds: {
						x: rect.x,
						y: rect.y,
						width: rect.width,
						height: rect.height
					},
					styles: {
						display: computed.display,
						position: computed.position,
						backgroundColor: computed.backgroundColor,
						color: computed.color,
						fontSize: computed.fontSize,
						padding: computed.padding,
						margin: computed.margin
					}
				};

				if (currentDepth < maxDepth && el.children.length > 0) {
					layout.children = [];
					for (const child of el.children) {
						layout.children.push(getLayout(child, currentDepth + 1, maxDepth));
					}
				}

				return layout;
			}

			const el = document.querySelector(%q);
			return getLayout(el, 0, %d);
		})()
	`, selector, depth)

	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return nil, err
	}

	if result.Value == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// Convert the result to ElementLayout
	jsonBytes, err := json.Marshal(result.Value)
	if err != nil {
		return nil, err
	}

	var layout ElementLayout
	if err := json.Unmarshal(jsonBytes, &layout); err != nil {
		return nil, err
	}

	layout.Selector = selector
	return &layout, nil
}

// ScreenshotElement captures a screenshot of a specific element.
func (c *Client) ScreenshotElement(ctx context.Context, targetID string, selector string, opts ScreenshotOptions) ([]byte, *BoundingBox, error) {
	// First get the bounding box
	bounds, err := c.GetBoundingBox(ctx, targetID, selector)
	if err != nil {
		return nil, nil, err
	}

	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, nil, err
	}

	// Take screenshot with clip region
	format := opts.Format
	if format == "" {
		format = "png"
	}

	params := map[string]interface{}{
		"format": format,
		"clip": map[string]interface{}{
			"x":      bounds.X,
			"y":      bounds.Y,
			"width":  bounds.Width,
			"height": bounds.Height,
			"scale":  1,
		},
	}

	if format == "jpeg" || format == "webp" {
		params["quality"] = opts.Quality
	}

	result, err := c.CallSession(ctx, sessionID, "Page.captureScreenshot", params)
	if err != nil {
		return nil, nil, err
	}

	var screenshot struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &screenshot); err != nil {
		return nil, nil, err
	}

	data, err := base64.StdEncoding.DecodeString(screenshot.Data)
	if err != nil {
		return nil, nil, err
	}

	return data, bounds, nil
}

// GetBoundingBox returns the bounding box of an element.
func (c *Client) GetBoundingBox(ctx context.Context, targetID string, selector string) (*BoundingBox, error) {
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
		})()
	`, selector)

	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return nil, err
	}
	if result.Value == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// Convert the map to BoundingBox
	m, ok := result.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	return &BoundingBox{
		X:      m["x"].(float64),
		Y:      m["y"].(float64),
		Width:  m["width"].(float64),
		Height: m["height"].(float64),
	}, nil
}

// SetViewport sets the browser viewport size.
func (c *Client) SetViewport(ctx context.Context, targetID string, width, height int) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Emulation.setDeviceMetricsOverride", map[string]interface{}{
		"width":             width,
		"height":            height,
		"deviceScaleFactor": 1,
		"mobile":            false,
	})
	if err != nil {
		return fmt.Errorf("setting viewport: %w", err)
	}

	return nil
}

// WaitForLoad waits for the page load event.
func (c *Client) WaitForLoad(ctx context.Context, targetID string) error {
	js := `
		new Promise((resolve) => {
			if (document.readyState === 'complete') {
				resolve(true);
			} else {
				window.addEventListener('load', () => resolve(true));
			}
		})
	`
	_, err := c.Eval(ctx, targetID, js)
	return err
}

// GetLocalStorage gets a value from localStorage.
func (c *Client) GetLocalStorage(ctx context.Context, targetID string, key string) (string, error) {
	js := fmt.Sprintf(`localStorage.getItem(%q)`, key)
	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return "", err
	}
	if result.Value == nil {
		return "", nil
	}
	if s, ok := result.Value.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// SetLocalStorage sets a value in localStorage.
func (c *Client) SetLocalStorage(ctx context.Context, targetID string, key, value string) error {
	js := fmt.Sprintf(`localStorage.setItem(%q, %q); true`, key, value)
	_, err := c.Eval(ctx, targetID, js)
	return err
}

// ClearLocalStorage clears all localStorage.
func (c *Client) ClearLocalStorage(ctx context.Context, targetID string) error {
	_, err := c.Eval(ctx, targetID, `localStorage.clear(); true`)
	return err
}

// GetSessionStorage gets a value from sessionStorage.
func (c *Client) GetSessionStorage(ctx context.Context, targetID string, key string) (string, error) {
	js := fmt.Sprintf(`sessionStorage.getItem(%q)`, key)
	result, err := c.Eval(ctx, targetID, js)
	if err != nil {
		return "", err
	}
	if result.Value == nil {
		return "", nil
	}
	if s, ok := result.Value.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// SetSessionStorage sets a value in sessionStorage.
func (c *Client) SetSessionStorage(ctx context.Context, targetID string, key, value string) error {
	js := fmt.Sprintf(`sessionStorage.setItem(%q, %q); true`, key, value)
	_, err := c.Eval(ctx, targetID, js)
	return err
}

// ClearSessionStorage clears all sessionStorage.
func (c *Client) ClearSessionStorage(ctx context.Context, targetID string) error {
	_, err := c.Eval(ctx, targetID, `sessionStorage.clear(); true`)
	return err
}

// NetworkConditions represents network throttling settings.
type NetworkConditions struct {
	Offline            bool    `json:"offline"`
	Latency            float64 `json:"latency"`             // Milliseconds
	DownloadThroughput float64 `json:"downloadThroughput"`  // Bytes per second (-1 = disabled)
	UploadThroughput   float64 `json:"uploadThroughput"`    // Bytes per second (-1 = disabled)
}

// NetworkPresets contains common network condition presets.
var NetworkPresets = map[string]NetworkConditions{
	"slow3g": {
		Offline:            false,
		Latency:            2000,
		DownloadThroughput: 50000,  // 50 KB/s
		UploadThroughput:   25000,  // 25 KB/s
	},
	"fast3g": {
		Offline:            false,
		Latency:            563,
		DownloadThroughput: 180000, // 180 KB/s
		UploadThroughput:   84375,  // ~84 KB/s
	},
	"4g": {
		Offline:            false,
		Latency:            170,
		DownloadThroughput: 1500000, // 1.5 MB/s
		UploadThroughput:   750000,  // 750 KB/s
	},
	"wifi": {
		Offline:            false,
		Latency:            28,
		DownloadThroughput: 3750000, // 3.75 MB/s
		UploadThroughput:   1875000, // 1.875 MB/s
	},
}

// EmulateNetworkConditions sets network throttling conditions.
func (c *Client) EmulateNetworkConditions(ctx context.Context, targetID string, conditions NetworkConditions) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Network domain first
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Network domain: %w", err)
	}

	params := map[string]interface{}{
		"offline":            conditions.Offline,
		"latency":            conditions.Latency,
		"downloadThroughput": conditions.DownloadThroughput,
		"uploadThroughput":   conditions.UploadThroughput,
	}

	_, err = c.CallSession(ctx, sessionID, "Network.emulateNetworkConditions", params)
	return err
}

// DisableNetworkThrottling disables network throttling.
func (c *Client) DisableNetworkThrottling(ctx context.Context, targetID string) error {
	return c.EmulateNetworkConditions(ctx, targetID, NetworkConditions{
		Offline:            false,
		Latency:            0,
		DownloadThroughput: -1, // Disabled
		UploadThroughput:   -1, // Disabled
	})
}

// HandleDialog sets up automatic dialog handling.
// action can be "accept" or "dismiss".
// promptText is the text to enter for prompts (optional).
func (c *Client) HandleDialog(ctx context.Context, targetID string, action string, promptText string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Page domain
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Page domain: %w", err)
	}

	// Subscribe to dialog events
	eventCh := c.subscribeEvent(sessionID, "Page.javascriptDialogOpening")

	// Handle dialog in background
	go func() {
		select {
		case <-eventCh:
			params := map[string]interface{}{
				"accept": action == "accept",
			}
			if promptText != "" {
				params["promptText"] = promptText
			}
			c.CallSession(ctx, sessionID, "Page.handleJavaScriptDialog", params)
		case <-ctx.Done():
		}
	}()

	return nil
}

// ExecuteScriptFile reads and executes JavaScript from a file.
func (c *Client) ExecuteScriptFile(ctx context.Context, targetID string, content string) (*EvalResult, error) {
	return c.Eval(ctx, targetID, content)
}

// RawCall sends a raw CDP command at the browser level.
// Returns the raw JSON response.
func (c *Client) RawCall(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	var p interface{}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params JSON: %w", err)
		}
	}
	return c.Call(ctx, method, p)
}

// RawCallSession sends a raw CDP command to a specific target/session.
// Returns the raw JSON response.
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

// CloseTab closes a browser tab by its target ID.
func (c *Client) CloseTab(ctx context.Context, targetID string) error {
	// Remove session from cache before closing
	c.sessionsMu.Lock()
	delete(c.sessions, targetID)
	c.sessionsMu.Unlock()

	_, err := c.Call(ctx, "Target.closeTarget", map[string]interface{}{
		"targetId": targetID,
	})
	if err != nil {
		return fmt.Errorf("closing target: %w", err)
	}
	return nil
}

// GetTitle returns the page title.
func (c *Client) GetTitle(ctx context.Context, targetID string) (string, error) {
	result, err := c.Eval(ctx, targetID, "document.title")
	if err != nil {
		return "", err
	}
	if result.Value == nil {
		return "", nil
	}
	if s, ok := result.Value.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// GetURL returns the current page URL.
func (c *Client) GetURL(ctx context.Context, targetID string) (string, error) {
	result, err := c.Eval(ctx, targetID, "document.location.href")
	if err != nil {
		return "", err
	}
	if result.Value == nil {
		return "", nil
	}
	if s, ok := result.Value.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// WaitForURL waits for the page URL to contain the given pattern.
func (c *Client) WaitForURL(ctx context.Context, targetID string, pattern string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		url, err := c.GetURL(ctx, targetID)
		if err != nil {
			return "", err
		}

		if strings.Contains(url, pattern) {
			return url, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return "", fmt.Errorf("timeout waiting for URL to contain %q", pattern)
}

// GetAttribute returns the value of an attribute for an element.
func (c *Client) GetAttribute(ctx context.Context, targetID string, selector string, name string) (string, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return "", err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return "", fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return "", fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return "", fmt.Errorf("parsing document response: %w", err)
	}

	// Query for element
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return "", fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return "", fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return "", fmt.Errorf("element not found: %s", selector)
	}

	// Get attributes using DOM.getAttributes
	attrResult, err := c.CallSession(ctx, sessionID, "DOM.getAttributes", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return "", fmt.Errorf("getting attributes: %w", err)
	}

	var attrResp struct {
		Attributes []string `json:"attributes"` // [name, value, name, value, ...]
	}
	if err := json.Unmarshal(attrResult, &attrResp); err != nil {
		return "", fmt.Errorf("parsing attributes response: %w", err)
	}

	// Find the attribute by name
	for i := 0; i < len(attrResp.Attributes)-1; i += 2 {
		if attrResp.Attributes[i] == name {
			return attrResp.Attributes[i+1], nil
		}
	}

	return "", nil // Attribute not found, return empty string
}

// DoubleClick double-clicks on an element specified by selector.
func (c *Client) DoubleClick(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query for element
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Get box model for coordinates
	boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("getting box model: %w", err)
	}

	var boxResp struct {
		Model struct {
			Content []float64 `json:"content"`
		} `json:"model"`
	}
	if err := json.Unmarshal(boxResult, &boxResp); err != nil {
		return fmt.Errorf("parsing box model response: %w", err)
	}

	content := boxResp.Model.Content
	if len(content) < 8 {
		return fmt.Errorf("invalid box model")
	}
	x := (content[0] + content[2] + content[4] + content[6]) / 4
	y := (content[1] + content[3] + content[5] + content[7]) / 4

	// Double-click: move, press, release, press, release with clickCount=2
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mousePressed: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseReleased: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 2,
	})
	if err != nil {
		return fmt.Errorf("dispatching mousePressed (2): %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 2,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseReleased (2): %w", err)
	}

	return nil
}

// RightClick right-clicks on an element specified by selector.
func (c *Client) RightClick(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("getting box model: %w", err)
	}

	var boxResp struct {
		Model struct {
			Content []float64 `json:"content"`
		} `json:"model"`
	}
	if err := json.Unmarshal(boxResult, &boxResp); err != nil {
		return fmt.Errorf("parsing box model response: %w", err)
	}

	content := boxResp.Model.Content
	if len(content) < 8 {
		return fmt.Errorf("invalid box model")
	}
	x := (content[0] + content[2] + content[4] + content[6]) / 4
	y := (content[1] + content[3] + content[5] + content[7]) / 4

	// Right-click: move, press right, release right
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "right",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mousePressed: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "right",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseReleased: %w", err)
	}

	return nil
}

// Clear clears a text input field.
func (c *Client) Clear(ctx context.Context, targetID string, selector string) error {
	// Focus the element first
	err := c.Focus(ctx, targetID, selector)
	if err != nil {
		return err
	}

	// Select all and delete
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Ctrl+A to select all
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type":                  "keyDown",
		"key":                   "a",
		"modifiers":             2, // Ctrl
		"windowsVirtualKeyCode": 65,
	})
	if err != nil {
		return fmt.Errorf("selecting all: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type":                  "keyUp",
		"key":                   "a",
		"modifiers":             2,
		"windowsVirtualKeyCode": 65,
	})
	if err != nil {
		return fmt.Errorf("selecting all (keyUp): %w", err)
	}

	// Delete key to clear
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type":                  "keyDown",
		"key":                   "Delete",
		"windowsVirtualKeyCode": 46,
	})
	if err != nil {
		return fmt.Errorf("deleting: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type":                  "keyUp",
		"key":                   "Delete",
		"windowsVirtualKeyCode": 46,
	})
	if err != nil {
		return fmt.Errorf("deleting (keyUp): %w", err)
	}

	return nil
}

// SelectOption selects an option in a <select> element by value.
func (c *Client) SelectOption(ctx context.Context, targetID string, selector string, value string) error {
	// Use JavaScript to select the option
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) throw new Error('Element not found');
			if (el.tagName !== 'SELECT') throw new Error('Element is not a select');
			el.value = %q;
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return el.value;
		})()
	`, selector, value)

	_, err := c.Eval(ctx, targetID, js)
	return err
}

// Check checks a checkbox or radio button.
func (c *Client) Check(ctx context.Context, targetID string, selector string) error {
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) throw new Error('Element not found');
			if (!el.checked) {
				el.checked = true;
				el.dispatchEvent(new Event('change', { bubbles: true }));
			}
			return el.checked;
		})()
	`, selector)

	_, err := c.Eval(ctx, targetID, js)
	return err
}

// Uncheck unchecks a checkbox.
func (c *Client) Uncheck(ctx context.Context, targetID string, selector string) error {
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) throw new Error('Element not found');
			if (el.checked) {
				el.checked = false;
				el.dispatchEvent(new Event('change', { bubbles: true }));
			}
			return !el.checked;
		})()
	`, selector)

	_, err := c.Eval(ctx, targetID, js)
	return err
}

// Hover moves the mouse over an element specified by selector.
func (c *Client) Hover(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query for element
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Get box model for element coordinates
	boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return fmt.Errorf("getting box model: %w", err)
	}

	var boxResp struct {
		Model struct {
			Content []float64 `json:"content"` // [x1,y1, x2,y2, x3,y3, x4,y4]
		} `json:"model"`
	}
	if err := json.Unmarshal(boxResult, &boxResp); err != nil {
		return fmt.Errorf("parsing box model response: %w", err)
	}

	// Calculate center point from content quad
	content := boxResp.Model.Content
	if len(content) < 8 {
		return fmt.Errorf("invalid box model")
	}
	x := (content[0] + content[2] + content[4] + content[6]) / 4
	y := (content[1] + content[3] + content[5] + content[7]) / 4

	// Dispatch mouse move event
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	return nil
}

// PrintToPDF generates a PDF of the page.
func (c *Client) PrintToPDF(ctx context.Context, targetID string, opts PDFOptions) ([]byte, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	params := make(map[string]interface{})
	if opts.Landscape {
		params["landscape"] = opts.Landscape
	}
	if opts.PrintBackground {
		params["printBackground"] = opts.PrintBackground
	}
	if opts.Scale > 0 {
		params["scale"] = opts.Scale
	}
	if opts.PaperWidth > 0 {
		params["paperWidth"] = opts.PaperWidth
	}
	if opts.PaperHeight > 0 {
		params["paperHeight"] = opts.PaperHeight
	}
	if opts.MarginTop > 0 {
		params["marginTop"] = opts.MarginTop
	}
	if opts.MarginBottom > 0 {
		params["marginBottom"] = opts.MarginBottom
	}
	if opts.MarginLeft > 0 {
		params["marginLeft"] = opts.MarginLeft
	}
	if opts.MarginRight > 0 {
		params["marginRight"] = opts.MarginRight
	}
	if opts.PageRanges != "" {
		params["pageRanges"] = opts.PageRanges
	}
	if opts.PreferCSSPageSize {
		params["preferCSSPageSize"] = opts.PreferCSSPageSize
	}

	result, err := c.CallSession(ctx, sessionID, "Page.printToPDF", params)
	if err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}

	var pdfResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &pdfResp); err != nil {
		return nil, fmt.Errorf("parsing PDF response: %w", err)
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(pdfResp.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding PDF data: %w", err)
	}

	return data, nil
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
	ID        int64           `json:"id"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *CDPError       `json:"error,omitempty"`
	Method    string          `json:"method,omitempty"`    // For events
	Params    json.RawMessage `json:"params,omitempty"`    // For events
	SessionID string          `json:"sessionId,omitempty"` // For session events
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

// subscribeEvent registers a handler for CDP events.
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

// InterceptConfig configures request/response interception.
type InterceptConfig struct {
	URLPattern        string            // URL pattern to match (e.g., "*", "*.js", "https://example.com/*")
	InterceptResponse bool              // If true, intercept responses; if false, intercept requests
	Replacements      map[string]string // Text replacements to apply (old -> new)
	ResponseBody      string            // Override response body entirely (if set, Replacements ignored)
	StatusCode        int               // Override status code (0 = use original)
	Headers           map[string]string // Override/add headers
}

// EnableIntercept enables request/response interception for the specified target.
func (c *Client) EnableIntercept(ctx context.Context, targetID string, config InterceptConfig) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Determine request stage
	requestStage := "Request"
	if config.InterceptResponse {
		requestStage = "Response"
	}

	// Build URL pattern
	urlPattern := config.URLPattern
	if urlPattern == "" {
		urlPattern = "*"
	}

	// Enable Fetch domain with patterns
	_, err = c.CallSession(ctx, sessionID, "Fetch.enable", map[string]interface{}{
		"patterns": []map[string]interface{}{
			{
				"urlPattern":   urlPattern,
				"requestStage": requestStage,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("enabling fetch: %w", err)
	}

	// Subscribe to requestPaused events
	eventCh := c.subscribeEvent(sessionID, "Fetch.requestPaused")

	// Handle events in a goroutine
	go func() {
		for params := range eventCh {
			var event struct {
				RequestID         string `json:"requestId"`
				Request           struct {
					URL string `json:"url"`
				} `json:"request"`
				ResponseStatusCode int               `json:"responseStatusCode"`
				ResponseHeaders    []json.RawMessage `json:"responseHeaders"`
			}
			if err := json.Unmarshal(params, &event); err != nil {
				continue
			}

			// For response interception, get and modify body
			if config.InterceptResponse {
				// Get response body
				result, err := c.CallSession(ctx, sessionID, "Fetch.getResponseBody", map[string]interface{}{
					"requestId": event.RequestID,
				})
				if err != nil {
					// Continue without modification on error
					c.CallSession(ctx, sessionID, "Fetch.continueRequest", map[string]interface{}{
						"requestId": event.RequestID,
					})
					continue
				}

				var bodyResult struct {
					Body          string `json:"body"`
					Base64Encoded bool   `json:"base64Encoded"`
				}
				if err := json.Unmarshal(result, &bodyResult); err != nil {
					c.CallSession(ctx, sessionID, "Fetch.continueRequest", map[string]interface{}{
						"requestId": event.RequestID,
					})
					continue
				}

				// Decode body if needed
				body := bodyResult.Body
				if bodyResult.Base64Encoded {
					decoded, err := base64.StdEncoding.DecodeString(body)
					if err == nil {
						body = string(decoded)
					}
				}

				// Apply modifications
				newBody := body
				if config.ResponseBody != "" {
					newBody = config.ResponseBody
				} else if len(config.Replacements) > 0 {
					for old, new := range config.Replacements {
						newBody = strings.ReplaceAll(newBody, old, new)
					}
				}

				// Determine status code
				statusCode := event.ResponseStatusCode
				if config.StatusCode > 0 {
					statusCode = config.StatusCode
				}

				// Build response headers
				responseHeaders := []map[string]string{}
				// Keep original headers (simplified - in real impl would parse responseHeaders)
				responseHeaders = append(responseHeaders, map[string]string{
					"name":  "Content-Type",
					"value": "text/html; charset=utf-8",
				})
				for name, value := range config.Headers {
					responseHeaders = append(responseHeaders, map[string]string{
						"name":  name,
						"value": value,
					})
				}

				// Fulfill with modified response
				c.CallSession(ctx, sessionID, "Fetch.fulfillRequest", map[string]interface{}{
					"requestId":       event.RequestID,
					"responseCode":    statusCode,
					"responseHeaders": responseHeaders,
					"body":            base64.StdEncoding.EncodeToString([]byte(newBody)),
				})
			} else {
				// For request interception, just continue (or modify request)
				c.CallSession(ctx, sessionID, "Fetch.continueRequest", map[string]interface{}{
					"requestId": event.RequestID,
				})
			}
		}
	}()

	return nil
}

// DisableIntercept disables request/response interception for the specified target.
func (c *Client) DisableIntercept(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Fetch.disable", nil)
	if err != nil {
		return fmt.Errorf("disabling fetch: %w", err)
	}

	return nil
}

// BlockURLs blocks network requests matching the specified URL patterns.
// Uses the Network.setBlockedURLs CDP method.
func (c *Client) BlockURLs(ctx context.Context, targetID string, patterns []string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Network domain first
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling network: %w", err)
	}

	// Set blocked URLs
	_, err = c.CallSession(ctx, sessionID, "Network.setBlockedURLs", map[string]interface{}{
		"urls": patterns,
	})
	if err != nil {
		return fmt.Errorf("setting blocked URLs: %w", err)
	}

	return nil
}

// UnblockURLs clears all URL blocking patterns.
func (c *Client) UnblockURLs(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Clear blocked URLs by setting empty array
	_, err = c.CallSession(ctx, sessionID, "Network.setBlockedURLs", map[string]interface{}{
		"urls": []string{},
	})
	if err != nil {
		return fmt.Errorf("clearing blocked URLs: %w", err)
	}

	return nil
}

// AccessibilityNode represents a node in the accessibility tree.
type AccessibilityNode struct {
	NodeID      string                 `json:"nodeId"`
	Role        string                 `json:"role"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Value       string                 `json:"value,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Children    []AccessibilityNode    `json:"children,omitempty"`
}

// WaitForNetworkIdle waits until there are no pending network requests for the specified duration.
func (c *Client) WaitForNetworkIdle(ctx context.Context, targetID string, idleTime time.Duration) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Network domain
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling network: %w", err)
	}

	// Subscribe to network events
	requestCh := c.subscribeEvent(sessionID, "Network.requestWillBeSent")
	responseCh := c.subscribeEvent(sessionID, "Network.loadingFinished")
	failedCh := c.subscribeEvent(sessionID, "Network.loadingFailed")

	defer c.unsubscribeEvent(sessionID, "Network.requestWillBeSent", requestCh)
	defer c.unsubscribeEvent(sessionID, "Network.loadingFinished", responseCh)
	defer c.unsubscribeEvent(sessionID, "Network.loadingFailed", failedCh)

	pendingRequests := make(map[string]bool)
	idleTimer := time.NewTimer(idleTime)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case params := <-requestCh:
			var event struct {
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(params, &event); err == nil {
				pendingRequests[event.RequestID] = true
				// Reset idle timer when new request starts
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(idleTime)
			}
		case params := <-responseCh:
			var event struct {
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(params, &event); err == nil {
				delete(pendingRequests, event.RequestID)
				// Reset idle timer when request finishes
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(idleTime)
			}
		case params := <-failedCh:
			var event struct {
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(params, &event); err == nil {
				delete(pendingRequests, event.RequestID)
				// Reset idle timer when request fails
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(idleTime)
			}
		case <-idleTimer.C:
			// No network activity for idleTime
			if len(pendingRequests) == 0 {
				return nil
			}
			// Still have pending requests, reset timer
			idleTimer.Reset(idleTime)
		}
	}
}

// GetPageSource returns the full HTML source of the page.
func (c *Client) GetPageSource(ctx context.Context, targetID string) (string, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return "", err
	}

	// Get the document root
	result, err := c.CallSession(ctx, sessionID, "DOM.getDocument", map[string]interface{}{
		"depth": -1,
	})
	if err != nil {
		return "", fmt.Errorf("getting document: %w", err)
	}

	var docResult struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(result, &docResult); err != nil {
		return "", fmt.Errorf("parsing document: %w", err)
	}

	// Get outer HTML of the root
	result, err = c.CallSession(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"nodeId": docResult.Root.NodeID,
	})
	if err != nil {
		return "", fmt.Errorf("getting outer HTML: %w", err)
	}

	var htmlResult struct {
		OuterHTML string `json:"outerHTML"`
	}
	if err := json.Unmarshal(result, &htmlResult); err != nil {
		return "", fmt.Errorf("parsing outer HTML: %w", err)
	}

	return htmlResult.OuterHTML, nil
}

// GetAccessibilityTree returns the accessibility tree for the page.
func (c *Client) GetAccessibilityTree(ctx context.Context, targetID string) ([]AccessibilityNode, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Accessibility domain
	_, err = c.CallSession(ctx, sessionID, "Accessibility.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling accessibility: %w", err)
	}

	// Get the full accessibility tree
	result, err := c.CallSession(ctx, sessionID, "Accessibility.getFullAXTree", nil)
	if err != nil {
		return nil, fmt.Errorf("getting accessibility tree: %w", err)
	}

	var treeResult struct {
		Nodes []struct {
			NodeID     string `json:"nodeId"`
			Role       struct {
				Value string `json:"value"`
			} `json:"role"`
			Name struct {
				Value string `json:"value"`
			} `json:"name"`
			Description struct {
				Value string `json:"value"`
			} `json:"description"`
			Value struct {
				Value string `json:"value"`
			} `json:"value"`
			Properties []struct {
				Name  string      `json:"name"`
				Value interface{} `json:"value"`
			} `json:"properties"`
			ChildIds []string `json:"childIds"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(result, &treeResult); err != nil {
		return nil, fmt.Errorf("parsing accessibility tree: %w", err)
	}

	// Convert to simpler format
	nodes := make([]AccessibilityNode, 0, len(treeResult.Nodes))
	for _, n := range treeResult.Nodes {
		// Skip ignored nodes
		if n.Role.Value == "none" || n.Role.Value == "ignored" {
			continue
		}

		node := AccessibilityNode{
			NodeID:      n.NodeID,
			Role:        n.Role.Value,
			Name:        n.Name.Value,
			Description: n.Description.Value,
			Value:       n.Value.Value,
		}

		if len(n.Properties) > 0 {
			node.Properties = make(map[string]interface{})
			for _, p := range n.Properties {
				node.Properties[p.Name] = p.Value
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetPerformanceMetrics returns performance metrics from the page.
func (c *Client) GetPerformanceMetrics(ctx context.Context, targetID string) (map[string]float64, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Performance domain
	_, err = c.CallSession(ctx, sessionID, "Performance.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling performance: %w", err)
	}

	// Get metrics
	result, err := c.CallSession(ctx, sessionID, "Performance.getMetrics", nil)
	if err != nil {
		return nil, fmt.Errorf("getting metrics: %w", err)
	}

	var metricsResult struct {
		Metrics []struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(result, &metricsResult); err != nil {
		return nil, fmt.Errorf("parsing metrics: %w", err)
	}

	metrics := make(map[string]float64)
	for _, m := range metricsResult.Metrics {
		metrics[m.Name] = m.Value
	}

	return metrics, nil
}

// SetOfflineMode enables or disables offline mode for network emulation.
func (c *Client) SetOfflineMode(ctx context.Context, targetID string, offline bool) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Network domain
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling network: %w", err)
	}

	// Set network conditions
	_, err = c.CallSession(ctx, sessionID, "Network.emulateNetworkConditions", map[string]interface{}{
		"offline":            offline,
		"latency":            0,
		"downloadThroughput": -1,
		"uploadThroughput":   -1,
	})
	if err != nil {
		return fmt.Errorf("setting offline mode: %w", err)
	}

	return nil
}

// MediaFeatures represents CSS media features to emulate.
type MediaFeatures struct {
	ColorScheme   string // "light", "dark", or "" for no preference
	ReducedMotion string // "reduce", "no-preference", or "" for no preference
	ForcedColors  string // "active", "none", or "" for no preference
}

// SetEmulatedMedia sets emulated media features.
func (c *Client) SetEmulatedMedia(ctx context.Context, targetID string, features MediaFeatures) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Build features array
	mediaFeatures := []map[string]interface{}{}

	if features.ColorScheme != "" {
		mediaFeatures = append(mediaFeatures, map[string]interface{}{
			"name":  "prefers-color-scheme",
			"value": features.ColorScheme,
		})
	}

	if features.ReducedMotion != "" {
		mediaFeatures = append(mediaFeatures, map[string]interface{}{
			"name":  "prefers-reduced-motion",
			"value": features.ReducedMotion,
		})
	}

	if features.ForcedColors != "" {
		mediaFeatures = append(mediaFeatures, map[string]interface{}{
			"name":  "forced-colors",
			"value": features.ForcedColors,
		})
	}

	_, err = c.CallSession(ctx, sessionID, "Emulation.setEmulatedMedia", map[string]interface{}{
		"features": mediaFeatures,
	})
	if err != nil {
		return fmt.Errorf("setting emulated media: %w", err)
	}

	return nil
}

// SetGeolocation overrides the geolocation for the specified target.
func (c *Client) SetGeolocation(ctx context.Context, targetID string, latitude, longitude, accuracy float64) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Emulation.setGeolocationOverride", map[string]interface{}{
		"latitude":  latitude,
		"longitude": longitude,
		"accuracy":  accuracy,
	})
	if err != nil {
		return fmt.Errorf("setting geolocation: %w", err)
	}

	return nil
}

// SetPermission sets the state of a permission for the page origin.
// permission: geolocation, notifications, midi, midi-sysex, push, camera, microphone, etc.
// state: granted, denied, prompt
func (c *Client) SetPermission(ctx context.Context, targetID string, permission, state string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Get the current URL to extract origin
	evalResult, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    "window.location.origin",
		"returnByValue": true,
	})
	if err != nil {
		return fmt.Errorf("getting page origin: %w", err)
	}

	var result struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(evalResult, &result); err != nil {
		return fmt.Errorf("parsing origin: %w", err)
	}

	origin := result.Result.Value
	if origin == "" || origin == "null" {
		return fmt.Errorf("page has no origin (navigate to a URL first)")
	}

	// Use Browser.setPermission (browser-level command)
	_, err = c.Call(ctx, "Browser.setPermission", map[string]interface{}{
		"permission": map[string]interface{}{
			"name": permission,
		},
		"setting": state,
		"origin":  origin,
	})
	if err != nil {
		return fmt.Errorf("setting permission: %w", err)
	}

	return nil
}

// WriteClipboard writes text to the clipboard.
func (c *Client) WriteClipboard(ctx context.Context, targetID string, text string) error {
	// First grant clipboard-write permission
	err := c.SetPermission(ctx, targetID, "clipboard-write", "granted")
	if err != nil {
		// Permission might fail on some origins, try anyway
	}

	js := fmt.Sprintf(`navigator.clipboard.writeText(%q).then(() => true)`, text)
	_, err = c.Eval(ctx, targetID, js)
	if err != nil {
		return fmt.Errorf("writing to clipboard: %w", err)
	}
	return nil
}

// ReadClipboard reads text from the clipboard.
func (c *Client) ReadClipboard(ctx context.Context, targetID string) (string, error) {
	// First grant clipboard-read permission
	err := c.SetPermission(ctx, targetID, "clipboard-read", "granted")
	if err != nil {
		// Permission might fail on some origins, try anyway
	}

	result, err := c.Eval(ctx, targetID, `navigator.clipboard.readText()`)
	if err != nil {
		return "", fmt.Errorf("reading from clipboard: %w", err)
	}

	if result.Value == nil {
		return "", nil
	}
	if s, ok := result.Value.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// SetUserAgent sets a custom user agent for the specified target.
func (c *Client) SetUserAgent(ctx context.Context, targetID string, userAgent string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Emulation.setUserAgentOverride", map[string]interface{}{
		"userAgent": userAgent,
	})
	if err != nil {
		return fmt.Errorf("setting user agent: %w", err)
	}

	return nil
}

// Emulate sets device emulation for the specified target.
func (c *Client) Emulate(ctx context.Context, targetID string, device DeviceInfo) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Set device metrics override
	_, err = c.CallSession(ctx, sessionID, "Emulation.setDeviceMetricsOverride", map[string]interface{}{
		"width":             device.Width,
		"height":            device.Height,
		"deviceScaleFactor": device.DeviceScaleFactor,
		"mobile":            device.Mobile,
	})
	if err != nil {
		return fmt.Errorf("setting device metrics: %w", err)
	}

	// Set user agent override
	_, err = c.CallSession(ctx, sessionID, "Emulation.setUserAgentOverride", map[string]interface{}{
		"userAgent": device.UserAgent,
	})
	if err != nil {
		return fmt.Errorf("setting user agent: %w", err)
	}

	return nil
}

// UploadFile sets files for a file input element.
func (c *Client) UploadFile(ctx context.Context, targetID string, selector string, files []string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Get document root
	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return fmt.Errorf("parsing document response: %w", err)
	}

	// Query selector
	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Set files on the input element
	_, err = c.CallSession(ctx, sessionID, "DOM.setFileInputFiles", map[string]interface{}{
		"nodeId": queryResp.NodeID,
		"files":  files,
	})
	if err != nil {
		return fmt.Errorf("setting files: %w", err)
	}

	return nil
}

// Exists checks if an element matching the selector exists.
func (c *Client) Exists(ctx context.Context, targetID string, selector string) (bool, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return false, err
	}

	// Enable Runtime domain
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return false, fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Use JavaScript to check if element exists
	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    fmt.Sprintf(`document.querySelector(%q) !== null`, selector),
		"returnByValue": true,
	})
	if err != nil {
		return false, fmt.Errorf("evaluating: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return false, fmt.Errorf("parsing eval response: %w", err)
	}

	return evalResp.Result.Value, nil
}

// WaitForNavigation waits for a navigation to complete.
func (c *Client) WaitForNavigation(ctx context.Context, targetID string, timeout time.Duration) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable Page domain
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Page domain: %w", err)
	}

	// Subscribe to frameNavigated event
	eventCh := c.subscribeEvent(sessionID, "Page.frameNavigated")
	defer c.unsubscribeEvent(sessionID, "Page.frameNavigated", eventCh)

	// Also subscribe to loadEventFired for full page load
	loadCh := c.subscribeEvent(sessionID, "Page.loadEventFired")
	defer c.unsubscribeEvent(sessionID, "Page.loadEventFired", loadCh)

	// Wait for either navigation or timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-loadCh:
		return nil
	case <-timeoutCtx.Done():
		return fmt.Errorf("timeout waiting for navigation")
	}
}

// GetValue retrieves the value of an input, textarea, or select element.
func (c *Client) GetValue(ctx context.Context, targetID string, selector string) (string, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return "", err
	}

	// Use JavaScript to get the value
	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf(`(function() {
			const el = document.querySelector(%q);
			if (!el) return {error: 'element not found'};
			return {value: el.value || ''};
		})()`, selector),
		"returnByValue": true,
	})
	if err != nil {
		return "", fmt.Errorf("getting value: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value struct {
				Error string `json:"error"`
				Value string `json:"value"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if evalResp.Result.Value.Error != "" {
		return "", fmt.Errorf("selector %q: %s", selector, evalResp.Result.Value.Error)
	}

	return evalResp.Result.Value.Value, nil
}

// WaitForFunction waits until a JavaScript expression evaluates to a truthy value.
func (c *Client) WaitForFunction(ctx context.Context, targetID string, expression string, timeout time.Duration) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 50 * time.Millisecond

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for function")
		}

		result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression":    expression,
			"returnByValue": true,
		})
		if err != nil {
			return fmt.Errorf("evaluating expression: %w", err)
		}

		var evalResp struct {
			Result struct {
				Value interface{} `json:"value"`
			} `json:"result"`
		}
		if err := json.Unmarshal(result, &evalResp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		// Check if value is truthy
		if isTruthy(evalResp.Result.Value) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}
}

// isTruthy checks if a value is truthy in JavaScript terms.
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case string:
		return val != ""
	case []interface{}:
		return true
	case map[string]interface{}:
		return true
	default:
		return true
	}
}

// FormInfo contains information about a form on the page.
type FormInfo struct {
	ID     string      `json:"id,omitempty"`
	Name   string      `json:"name,omitempty"`
	Action string      `json:"action,omitempty"`
	Method string      `json:"method,omitempty"`
	Inputs []InputInfo `json:"inputs"`
}

// InputInfo contains information about a form input.
type InputInfo struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type"`
	ID          string `json:"id,omitempty"`
	Value       string `json:"value,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// GetForms returns information about all forms on the page.
func (c *Client) GetForms(ctx context.Context, targetID string) ([]FormInfo, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `(function() {
			const forms = [];
			document.querySelectorAll('form').forEach(form => {
				const inputs = [];
				form.querySelectorAll('input, textarea, select').forEach(input => {
					inputs.push({
						name: input.name || '',
						type: input.type || input.tagName.toLowerCase(),
						id: input.id || '',
						value: input.value || '',
						placeholder: input.placeholder || '',
						required: input.required || false
					});
				});
				forms.push({
					id: form.id || '',
					name: form.name || '',
					action: form.action || '',
					method: form.method || 'get',
					inputs: inputs
				});
			});
			return forms;
		})()`,
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting forms: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value []FormInfo `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return evalResp.Result.Value, nil
}

// Highlight adds a visual highlight to an element for debugging.
func (c *Client) Highlight(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Enable DOM domain
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling DOM: %w", err)
	}

	// Get document root
	result, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(result, &docResp); err != nil {
		return fmt.Errorf("parsing document: %w", err)
	}

	// Query selector
	result, err = c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &queryResp); err != nil {
		return fmt.Errorf("parsing query response: %w", err)
	}
	if queryResp.NodeID == 0 {
		return fmt.Errorf("selector %q: element not found", selector)
	}

	// Highlight the node using Overlay domain
	_, err = c.CallSession(ctx, sessionID, "Overlay.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Overlay: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Overlay.highlightNode", map[string]interface{}{
		"nodeId": queryResp.NodeID,
		"highlightConfig": map[string]interface{}{
			"showInfo":       true,
			"showExtensions": true,
			"contentColor":   map[string]interface{}{"r": 111, "g": 168, "b": 220, "a": 0.66},
			"paddingColor":   map[string]interface{}{"r": 147, "g": 196, "b": 125, "a": 0.55},
			"borderColor":    map[string]interface{}{"r": 255, "g": 229, "b": 153, "a": 0.66},
			"marginColor":    map[string]interface{}{"r": 246, "g": 178, "b": 107, "a": 0.66},
		},
	})
	if err != nil {
		return fmt.Errorf("highlighting node: %w", err)
	}

	return nil
}

// HideHighlight removes any element highlight.
func (c *Client) HideHighlight(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Overlay.hideHighlight", nil)
	if err != nil {
		return fmt.Errorf("hiding highlight: %w", err)
	}

	return nil
}

// ImageInfo contains information about an image element.
type ImageInfo struct {
	Src     string `json:"src"`
	Alt     string `json:"alt,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Loading string `json:"loading,omitempty"`
}

// GetImages returns all images on the page.
func (c *Client) GetImages(ctx context.Context, targetID string) ([]ImageInfo, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `(function() {
			const images = [];
			document.querySelectorAll('img').forEach(img => {
				images.push({
					src: img.src || '',
					alt: img.alt || '',
					width: img.naturalWidth || img.width || 0,
					height: img.naturalHeight || img.height || 0,
					loading: img.loading || ''
				});
			});
			return images;
		})()`,
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting images: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value []ImageInfo `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return evalResp.Result.Value, nil
}

// ScrollToBottom scrolls to the bottom of the page.
func (c *Client) ScrollToBottom(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `window.scrollTo(0, document.body.scrollHeight)`,
	})
	if err != nil {
		return fmt.Errorf("scrolling to bottom: %w", err)
	}

	return nil
}

// ScrollToTop scrolls to the top of the page.
func (c *Client) ScrollToTop(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `window.scrollTo(0, 0)`,
	})
	if err != nil {
		return fmt.Errorf("scrolling to top: %w", err)
	}

	return nil
}

// FrameInfo contains information about a frame.
type FrameInfo struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url"`
}

// frameTreeNode represents a node in the frame tree (used for recursive parsing).
type frameTreeNode struct {
	Frame struct {
		ID       string `json:"id"`
		ParentID string `json:"parentId"`
		Name     string `json:"name"`
		URL      string `json:"url"`
	} `json:"frame"`
	ChildFrames []frameTreeNode `json:"childFrames"`
}

// collectFrames recursively collects all frames from the frame tree.
func collectFrames(node frameTreeNode, frames *[]FrameInfo) {
	*frames = append(*frames, FrameInfo{
		ID:       node.Frame.ID,
		ParentID: node.Frame.ParentID,
		Name:     node.Frame.Name,
		URL:      node.Frame.URL,
	})
	for _, child := range node.ChildFrames {
		collectFrames(child, frames)
	}
}

// GetFrames returns all frames in the page (including nested iframes).
func (c *Client) GetFrames(ctx context.Context, targetID string) ([]FrameInfo, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Page domain
	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Page domain: %w", err)
	}

	// Get frame tree
	result, err := c.CallSession(ctx, sessionID, "Page.getFrameTree", nil)
	if err != nil {
		return nil, fmt.Errorf("getting frame tree: %w", err)
	}

	var resp struct {
		FrameTree frameTreeNode `json:"frameTree"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing frame tree: %w", err)
	}

	// Collect all frames recursively
	var frames []FrameInfo
	collectFrames(resp.FrameTree, &frames)

	return frames, nil
}

// EvalInFrame evaluates JavaScript in a specific frame.
func (c *Client) EvalInFrame(ctx context.Context, targetID string, frameID string, expression string) (*EvalResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Create isolated world for the frame to execute in
	result, err := c.CallSession(ctx, sessionID, "Page.createIsolatedWorld", map[string]interface{}{
		"frameId": frameID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating isolated world: %w", err)
	}

	var worldResp struct {
		ExecutionContextID int64 `json:"executionContextId"`
	}
	if err := json.Unmarshal(result, &worldResp); err != nil {
		return nil, fmt.Errorf("parsing world response: %w", err)
	}

	// Execute in that context
	result, err = c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":       expression,
		"contextId":        worldResp.ExecutionContextID,
		"returnByValue":    true,
		"awaitPromise":     true,
		"userGesture":      true,
		"replMode":         false,
		"allowUnsafeEvalBlockedByCSP": false,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluating in frame: %w", err)
	}

	var evalResp struct {
		Result struct {
			Type        string      `json:"type"`
			Value       interface{} `json:"value"`
			Description string      `json:"description,omitempty"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails,omitempty"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing eval response: %w", err)
	}

	if evalResp.ExceptionDetails != nil {
		return nil, fmt.Errorf("JS error: %s", evalResp.ExceptionDetails.Text)
	}

	return &EvalResult{
		Type:  evalResp.Result.Type,
		Value: evalResp.Result.Value,
	}, nil
}

// WaitForGone waits for an element to be removed from the DOM.
func (c *Client) WaitForGone(ctx context.Context, targetID string, selector string, timeout time.Duration) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 50 * time.Millisecond

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %q to be removed", selector)
		}

		// Check if element exists
		result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression":    fmt.Sprintf(`document.querySelector(%q) === null`, selector),
			"returnByValue": true,
		})
		if err != nil {
			return fmt.Errorf("checking selector: %w", err)
		}

		var evalResp struct {
			Result struct {
				Value bool `json:"value"`
			} `json:"result"`
		}
		if err := json.Unmarshal(result, &evalResp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		// Element is gone
		if evalResp.Result.Value {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}
}

// WaitRequestResult contains the result of waiting for a network request.
type WaitRequestResult struct {
	Found     bool   `json:"found"`
	URL       string `json:"url"`
	Method    string `json:"method"`
	RequestID string `json:"requestId"`
}

// WaitForRequest waits for a network request with a URL containing the pattern.
func (c *Client) WaitForRequest(ctx context.Context, targetID string, pattern string, timeout time.Duration) (*WaitRequestResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Network domain
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling network: %w", err)
	}

	// Subscribe to request events
	requestCh := c.subscribeEvent(sessionID, "Network.requestWillBeSent")
	defer c.unsubscribeEvent(sessionID, "Network.requestWillBeSent", requestCh)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("timeout waiting for request matching %q", pattern)
			}
			return nil, timeoutCtx.Err()
		case params, ok := <-requestCh:
			if !ok {
				return nil, fmt.Errorf("event channel closed")
			}
			var event struct {
				RequestID string `json:"requestId"`
				Request   struct {
					URL    string `json:"url"`
					Method string `json:"method"`
				} `json:"request"`
			}
			if err := json.Unmarshal(params, &event); err != nil {
				continue
			}
			// Check if URL contains the pattern
			if strings.Contains(event.Request.URL, pattern) {
				return &WaitRequestResult{
					Found:     true,
					URL:       event.Request.URL,
					Method:    event.Request.Method,
					RequestID: event.RequestID,
				}, nil
			}
		}
	}
}

// WaitResponseResult contains the result of waiting for a network response.
type WaitResponseResult struct {
	Found     bool   `json:"found"`
	URL       string `json:"url"`
	Status    int    `json:"status"`
	MimeType  string `json:"mimeType,omitempty"`
	RequestID string `json:"requestId"`
}

// ComputedStyleResult contains the computed style value for an element.
type ComputedStyleResult struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

// WaitForResponse waits for a network response with a URL containing the pattern.
func (c *Client) WaitForResponse(ctx context.Context, targetID string, pattern string, timeout time.Duration) (*WaitResponseResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Network domain
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling network: %w", err)
	}

	// Subscribe to response events
	responseCh := c.subscribeEvent(sessionID, "Network.responseReceived")
	defer c.unsubscribeEvent(sessionID, "Network.responseReceived", responseCh)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("timeout waiting for response matching %q", pattern)
			}
			return nil, timeoutCtx.Err()
		case params, ok := <-responseCh:
			if !ok {
				return nil, fmt.Errorf("event channel closed")
			}
			var event struct {
				RequestID string `json:"requestId"`
				Response  struct {
					URL      string `json:"url"`
					Status   int    `json:"status"`
					MimeType string `json:"mimeType"`
				} `json:"response"`
			}
			if err := json.Unmarshal(params, &event); err != nil {
				continue
			}
			// Check if URL contains the pattern
			if strings.Contains(event.Response.URL, pattern) {
				return &WaitResponseResult{
					Found:     true,
					URL:       event.Response.URL,
					Status:    event.Response.Status,
					MimeType:  event.Response.MimeType,
					RequestID: event.RequestID,
				}, nil
			}
		}
	}
}

// GetComputedStyle returns the computed style value for a CSS property of an element.
func (c *Client) GetComputedStyle(ctx context.Context, targetID string, selector string, property string) (*ComputedStyleResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Use JavaScript to get the computed style
	jsExpr := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) {
				return {error: 'element not found'};
			}
			const style = window.getComputedStyle(el);
			return {value: style.getPropertyValue(%q)};
		})()
	`, selector, property)

	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    jsExpr,
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluating computed style: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value struct {
				Error string `json:"error"`
				Value string `json:"value"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if evalResp.Result.Value.Error != "" {
		return nil, fmt.Errorf("%s", evalResp.Result.Value.Error)
	}

	return &ComputedStyleResult{
		Property: property,
		Value:    evalResp.Result.Value.Value,
	}, nil
}
