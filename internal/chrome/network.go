package chrome

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

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
	har.Log.Creator = HARCreator{Name: "hubcap", Version: "1.0"}
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

// GetCSSCoverage captures CSS rule usage data.
func (c *Client) GetCSSCoverage(ctx context.Context, targetID string) (*CSSCoverageResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable DOM domain first (required by CSS domain)
	_, err = c.CallSession(ctx, sessionID, "DOM.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling DOM domain: %w", err)
	}

	// Enable CSS domain
	_, err = c.CallSession(ctx, sessionID, "CSS.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling CSS domain: %w", err)
	}

	// Start rule usage tracking
	_, err = c.CallSession(ctx, sessionID, "CSS.startRuleUsageTracking", nil)
	if err != nil {
		return nil, fmt.Errorf("starting rule usage tracking: %w", err)
	}

	// Take coverage delta
	result, err := c.CallSession(ctx, sessionID, "CSS.takeCoverageDelta", nil)
	if err != nil {
		return nil, fmt.Errorf("taking coverage delta: %w", err)
	}

	// Stop tracking and disable
	c.CallSession(ctx, sessionID, "CSS.stopRuleUsageTracking", nil)
	c.CallSession(ctx, sessionID, "CSS.disable", nil)

	var resp struct {
		Coverage []struct {
			StyleSheetID string `json:"styleSheetId"`
			StartOffset  int    `json:"startOffset"`
			EndOffset    int    `json:"endOffset"`
			Used         bool   `json:"used"`
		} `json:"coverage"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing coverage response: %w", err)
	}

	entries := make([]CSSCoverageEntry, len(resp.Coverage))
	for i, c := range resp.Coverage {
		entries[i] = CSSCoverageEntry{
			StyleSheetID: c.StyleSheetID,
			StartOffset:  c.StartOffset,
			EndOffset:    c.EndOffset,
			Used:         c.Used,
		}
	}

	return &CSSCoverageResult{Entries: entries}, nil
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
// Uses the Network.setBlockedURLs protocol method.
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

// GetResponseBody retrieves the response body for a network request by its request ID.
func (c *Client) GetResponseBody(ctx context.Context, targetID string, requestID string) (*ResponseBodyResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Enable Network domain
	_, err = c.CallSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("enabling network: %w", err)
	}

	result, err := c.CallSession(ctx, sessionID, "Network.getResponseBody", map[string]interface{}{
		"requestId": requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting response body: %w", err)
	}

	var resp struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing response body: %w", err)
	}

	return &ResponseBodyResult{
		Body:          resp.Body,
		Base64Encoded: resp.Base64Encoded,
	}, nil
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
