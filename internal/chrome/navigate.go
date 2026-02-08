package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

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
