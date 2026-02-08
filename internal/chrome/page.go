package chrome

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

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
		"expression":                  expression,
		"contextId":                   worldResp.ExecutionContextID,
		"returnByValue":               true,
		"awaitPromise":                true,
		"userGesture":                 true,
		"replMode":                    false,
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
			NodeID string `json:"nodeId"`
			Role   struct {
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

// GetDOMSnapshot captures a full serialized DOM tree.
func (c *Client) GetDOMSnapshot(ctx context.Context, targetID string) (*DOMSnapshotResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	result, err := c.CallSession(ctx, sessionID, "DOMSnapshot.captureSnapshot", map[string]interface{}{
		"computedStyles": []string{"display", "visibility", "opacity"},
	})
	if err != nil {
		return nil, fmt.Errorf("capturing DOM snapshot: %w", err)
	}

	var resp struct {
		Documents []json.RawMessage `json:"documents"`
		Strings   []string          `json:"strings"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing snapshot response: %w", err)
	}

	return &DOMSnapshotResult{
		Documents: resp.Documents,
		Strings:   resp.Strings,
	}, nil
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

// ExecuteScriptFile reads and executes JavaScript from a file.
func (c *Client) ExecuteScriptFile(ctx context.Context, targetID string, content string) (*EvalResult, error) {
	return c.Eval(ctx, targetID, content)
}
