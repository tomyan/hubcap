package chrome

import (
	"context"
	"encoding/json"
)

// RecordedEvent represents a recorded browser event.
type RecordedEvent struct {
	Type string `json:"type"` // "navigate", "click", etc.
	URL  string `json:"url,omitempty"`
}

// RecordNavigations subscribes to Page.frameNavigated events and sends
// top-level navigation URLs to the returned channel. The channel is closed
// when the context is cancelled or the connection drops.
func (c *Client) RecordNavigations(ctx context.Context, targetID string) (<-chan RecordedEvent, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	_, err = c.CallSession(ctx, sessionID, "Page.enable", nil)
	if err != nil {
		return nil, err
	}

	navCh := c.subscribeEvent(sessionID, "Page.frameNavigated")

	out := make(chan RecordedEvent, 16)

	go func() {
		defer close(out)
		defer c.unsubscribeEvent(sessionID, "Page.frameNavigated", navCh)

		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-navCh:
				if !ok {
					return
				}
				var params struct {
					Frame struct {
						URL      string `json:"url"`
						ParentID string `json:"parentId"`
					} `json:"frame"`
				}
				if err := json.Unmarshal(raw, &params); err != nil {
					continue
				}
				if params.Frame.ParentID != "" {
					continue
				}
				if params.Frame.URL == "" || params.Frame.URL == "about:blank" {
					continue
				}
				select {
				case out <- RecordedEvent{Type: "navigate", URL: params.Frame.URL}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}
