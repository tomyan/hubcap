package cdp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
func (c *Client) attachToTarget(ctx context.Context, targetID string) (string, error) {
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

// PressKey presses a special key (Enter, Tab, Escape, etc.).
func (c *Client) PressKey(ctx context.Context, targetID string, key string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Get key code if available
	keyCode, hasKeyCode := keyCodeMap[key]

	params := map[string]interface{}{
		"type": "keyDown",
		"key":  key,
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
// Returns a channel that receives messages. The channel is buffered.
// The caller should read from the channel to receive messages.
func (c *Client) CaptureConsole(ctx context.Context, targetID string) (<-chan ConsoleMessage, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Runtime domain to receive console events
	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Runtime domain: %w", err)
	}

	// Subscribe to console API events
	eventCh := c.subscribeEvent(sessionID, "Runtime.consoleAPICalled")

	// Create output channel
	output := make(chan ConsoleMessage, 100)

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
			case <-c.closeCh:
				return
			}
		}
	}()

	return output, nil
}

// CaptureNetwork starts capturing network events from a page.
// Returns a channel that receives NetworkEvent. The channel is buffered.
func (c *Client) CaptureNetwork(ctx context.Context, targetID string) (<-chan NetworkEvent, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Network domain to receive events
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling Network domain: %w", err)
	}

	// Subscribe to network events
	requestCh := c.subscribeEvent(sessionID, "Network.requestWillBeSent")
	responseCh := c.subscribeEvent(sessionID, "Network.responseReceived")

	// Create output channel
	output := make(chan NetworkEvent, 100)

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
			case <-c.closeCh:
				return
			}
		}
	}()

	return output, nil
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
