package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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
