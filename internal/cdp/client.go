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
