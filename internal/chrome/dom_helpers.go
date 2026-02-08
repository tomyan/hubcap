package chrome

import (
	"context"
	"encoding/json"
	"fmt"
)

// resolveNodeID enables DOM, gets the document root, and runs querySelector
// to find the first element matching selector. Returns the node ID.
func (c *Client) resolveNodeID(ctx context.Context, sessionID string, selector string) (int64, error) {
	_, err := c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return 0, fmt.Errorf("enabling DOM domain: %w", err)
	}

	docResult, err := c.CallSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return 0, fmt.Errorf("getting document: %w", err)
	}

	var docResp struct {
		Root struct {
			NodeID int64 `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return 0, fmt.Errorf("parsing document response: %w", err)
	}

	queryResult, err := c.CallSession(ctx, sessionID, "DOM.querySelector", map[string]interface{}{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return 0, fmt.Errorf("querying selector: %w", err)
	}

	var queryResp struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return 0, fmt.Errorf("parsing query response: %w", err)
	}

	if queryResp.NodeID == 0 {
		return 0, fmt.Errorf("element not found: %s", selector)
	}

	return queryResp.NodeID, nil
}

// resolveElementCenter finds an element by selector and returns its center coordinates.
func (c *Client) resolveElementCenter(ctx context.Context, sessionID string, selector string) (x, y float64, err error) {
	nodeID, err := c.resolveNodeID(ctx, sessionID, selector)
	if err != nil {
		return 0, 0, err
	}

	return c.getNodeCenter(ctx, sessionID, nodeID)
}

// getNodeCenter returns the center coordinates of a DOM node by its node ID.
func (c *Client) getNodeCenter(ctx context.Context, sessionID string, nodeID int64) (x, y float64, err error) {
	boxResult, err := c.CallSession(ctx, sessionID, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": nodeID,
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
		return 0, 0, fmt.Errorf("parsing box model response: %w", err)
	}

	content := boxResp.Model.Content
	if len(content) < 8 {
		return 0, 0, fmt.Errorf("invalid box model")
	}

	x = (content[0] + content[2] + content[4] + content[6]) / 4
	y = (content[1] + content[3] + content[5] + content[7]) / 4
	return x, y, nil
}

// dispatchMouseClick dispatches mouseMoved, mousePressed, and mouseReleased events.
func (c *Client) dispatchMouseClick(ctx context.Context, sessionID string, x, y float64, button string, clickCount int) error {
	_, err := c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
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
		"button":     button,
		"clickCount": clickCount,
	})
	if err != nil {
		return fmt.Errorf("dispatching mousePressed: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     button,
		"clickCount": clickCount,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseReleased: %w", err)
	}

	return nil
}
