package chrome

import (
	"context"
	"encoding/json"
	"fmt"
)

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
