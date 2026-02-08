package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

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

	// Parse attributes (Chrome returns flat array: [name, value, name, value, ...])
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

// GetSelection returns the currently selected text on the page.
func (c *Client) GetSelection(ctx context.Context, targetID string) (*SelectionResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    "window.getSelection().toString()",
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting selection: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &SelectionResult{
		Text: evalResp.Result.Value,
	}, nil
}

// GetCaretPosition returns the caret (cursor) position in an input or textarea element.
func (c *Client) GetCaretPosition(ctx context.Context, targetID string, selector string) (*CaretPositionResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Use JavaScript to get caret position
	jsExpr := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) {
				return {error: 'element not found'};
			}
			if (typeof el.selectionStart !== 'number') {
				return {error: 'element does not support selection'};
			}
			return {start: el.selectionStart, end: el.selectionEnd};
		})()
	`, selector)

	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    jsExpr,
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting caret position: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value struct {
				Error string `json:"error"`
				Start int    `json:"start"`
				End   int    `json:"end"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if evalResp.Result.Value.Error != "" {
		return nil, fmt.Errorf("%s", evalResp.Result.Value.Error)
	}

	return &CaretPositionResult{
		Start: evalResp.Result.Value.Start,
		End:   evalResp.Result.Value.End,
	}, nil
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

// GetEventListeners returns the event listeners attached to a DOM element.
func (c *Client) GetEventListeners(ctx context.Context, targetID string, selector string) (*EventListenersResult, error) {
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

	// Query for element
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
	if queryResp.NodeID == 0 {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// Resolve node to get remote object ID
	resolveResult, err := c.CallSession(ctx, sessionID, "DOM.resolveNode", map[string]interface{}{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving node: %w", err)
	}

	var resolveResp struct {
		Object struct {
			ObjectID string `json:"objectId"`
		} `json:"object"`
	}
	if err := json.Unmarshal(resolveResult, &resolveResp); err != nil {
		return nil, fmt.Errorf("parsing resolve response: %w", err)
	}

	// Get event listeners using DOMDebugger
	listenersResult, err := c.CallSession(ctx, sessionID, "DOMDebugger.getEventListeners", map[string]interface{}{
		"objectId": resolveResp.Object.ObjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting event listeners: %w", err)
	}

	var listenersResp struct {
		Listeners []struct {
			Type         string `json:"type"`
			UseCapture   bool   `json:"useCapture"`
			Passive      bool   `json:"passive"`
			Once         bool   `json:"once"`
			ScriptID     string `json:"scriptId"`
			LineNumber   int    `json:"lineNumber"`
			ColumnNumber int    `json:"columnNumber"`
		} `json:"listeners"`
	}
	if err := json.Unmarshal(listenersResult, &listenersResp); err != nil {
		return nil, fmt.Errorf("parsing listeners response: %w", err)
	}

	elResult := &EventListenersResult{
		Listeners: make([]EventListenerInfo, len(listenersResp.Listeners)),
	}
	for i, l := range listenersResp.Listeners {
		elResult.Listeners[i] = EventListenerInfo{
			Type:         l.Type,
			UseCapture:   l.UseCapture,
			Passive:      l.Passive,
			Once:         l.Once,
			ScriptID:     l.ScriptID,
			LineNumber:   l.LineNumber,
			ColumnNumber: l.ColumnNumber,
		}
	}

	return elResult, nil
}
