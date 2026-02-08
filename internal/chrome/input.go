package chrome

import (
	"context"
	"encoding/json"
	"fmt"
)

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

// Click clicks on the first element matching a CSS selector.
func (c *Client) Click(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	x, y, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	return c.dispatchMouseClick(ctx, sessionID, x, y, "left", 1)
}

// ClickAt clicks at specific x, y coordinates.
func (c *Client) ClickAt(ctx context.Context, targetID string, x, y float64) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	return c.dispatchMouseClick(ctx, sessionID, x, y, "left", 1)
}

// DoubleClick double-clicks on an element specified by selector.
func (c *Client) DoubleClick(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	x, y, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	// Double-click: move, press(1), release(1), press(2), release(2)
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	for _, clickCount := range []int{1, 2} {
		_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
			"type":       "mousePressed",
			"x":          x,
			"y":          y,
			"button":     "left",
			"clickCount": clickCount,
		})
		if err != nil {
			return fmt.Errorf("dispatching mousePressed (%d): %w", clickCount, err)
		}

		_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
			"type":       "mouseReleased",
			"x":          x,
			"y":          y,
			"button":     "left",
			"clickCount": clickCount,
		})
		if err != nil {
			return fmt.Errorf("dispatching mouseReleased (%d): %w", clickCount, err)
		}
	}

	return nil
}

// RightClick right-clicks on an element specified by selector.
func (c *Client) RightClick(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	x, y, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	return c.dispatchMouseClick(ctx, sessionID, x, y, "right", 1)
}

// TripleClick triple-clicks on an element specified by selector.
// This is typically used to select an entire paragraph.
func (c *Client) TripleClick(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	x, y, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	// Move mouse to element
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return fmt.Errorf("dispatching mouseMoved: %w", err)
	}

	for _, clickCount := range []int{1, 2, 3} {
		_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
			"type":       "mousePressed",
			"x":          x,
			"y":          y,
			"button":     "left",
			"clickCount": clickCount,
		})
		if err != nil {
			return fmt.Errorf("dispatching mousePressed (%d): %w", clickCount, err)
		}

		_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
			"type":       "mouseReleased",
			"x":          x,
			"y":          y,
			"button":     "left",
			"clickCount": clickCount,
		})
		if err != nil {
			return fmt.Errorf("dispatching mouseReleased (%d): %w", clickCount, err)
		}
	}

	return nil
}

// Tap performs a touch tap on an element (like a finger tap on mobile).
func (c *Client) Tap(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	x, y, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type": "touchStart",
		"touchPoints": []map[string]interface{}{
			{"x": x, "y": y},
		},
	})
	if err != nil {
		return fmt.Errorf("dispatching touchStart: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type":        "touchEnd",
		"touchPoints": []map[string]interface{}{},
	})
	if err != nil {
		return fmt.Errorf("dispatching touchEnd: %w", err)
	}

	return nil
}

// Drag performs a drag from one element to another.
func (c *Client) Drag(ctx context.Context, targetID string, sourceSelector, destSelector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	srcX, srcY, err := c.resolveElementCenter(ctx, sessionID, sourceSelector)
	if err != nil {
		return err
	}
	dstX, dstY, err := c.resolveElementCenter(ctx, sessionID, destSelector)
	if err != nil {
		return err
	}

	// Perform drag: move to source, press, move to dest, release
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    srcX,
		"y":    srcY,
	})
	if err != nil {
		return fmt.Errorf("moving to source: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"x":          srcX,
		"y":          srcY,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("pressing at source: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    dstX,
		"y":    dstY,
	})
	if err != nil {
		return fmt.Errorf("moving to destination: %w", err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"x":          dstX,
		"y":          dstY,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return fmt.Errorf("releasing at destination: %w", err)
	}

	return nil
}

// Fill fills an input element with text.
func (c *Client) Fill(ctx context.Context, targetID string, selector string, text string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	nodeID, err := c.resolveNodeID(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	// Focus the element
	_, err = c.CallSession(ctx, sessionID, "DOM.focus", map[string]interface{}{
		"nodeId": nodeID,
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

	// Process text with escape sequence support
	i := 0
	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) {
			next := text[i+1]
			switch next {
			case 'n':
				// \n → Enter key
				if err := c.typeSpecialKey(ctx, sessionID, "Enter", "\r", 13); err != nil {
					return err
				}
				i += 2
				continue
			case 't':
				// \t → Tab key
				if err := c.typeSpecialKey(ctx, sessionID, "Tab", "", 9); err != nil {
					return err
				}
				i += 2
				continue
			case '\\':
				// \\ → literal backslash
				if err := c.typeChar(ctx, sessionID, `\`); err != nil {
					return err
				}
				i += 2
				continue
			}
		}

		// Regular character
		charStr := string(text[i])
		if err := c.typeChar(ctx, sessionID, charStr); err != nil {
			return err
		}
		i++
	}

	return nil
}

// typeChar dispatches key events for a regular character.
func (c *Client) typeChar(ctx context.Context, sessionID string, char string) error {
	_, err := c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type": "keyDown",
		"text": char,
		"key":  char,
	})
	if err != nil {
		return fmt.Errorf("keyDown for %q: %w", char, err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type": "keyUp",
		"key":  char,
	})
	if err != nil {
		return fmt.Errorf("keyUp for %q: %w", char, err)
	}
	return nil
}

// typeSpecialKey dispatches key events for a special key (Enter, Tab, etc).
func (c *Client) typeSpecialKey(ctx context.Context, sessionID string, key string, text string, keyCode int) error {
	params := map[string]interface{}{
		"type":                  "keyDown",
		"key":                   key,
		"windowsVirtualKeyCode": keyCode,
		"nativeVirtualKeyCode":  keyCode,
	}
	if text != "" {
		params["text"] = text
	}
	_, err := c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", params)
	if err != nil {
		return fmt.Errorf("keyDown for %q: %w", key, err)
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchKeyEvent", map[string]interface{}{
		"type":                  "keyUp",
		"key":                   key,
		"windowsVirtualKeyCode": keyCode,
		"nativeVirtualKeyCode":  keyCode,
	})
	if err != nil {
		return fmt.Errorf("keyUp for %q: %w", key, err)
	}
	return nil
}

// PressKey presses a special key (Enter, Tab, Escape, etc.).
func (c *Client) PressKey(ctx context.Context, targetID string, key string) error {
	return c.PressKeyWithModifiers(ctx, targetID, key, KeyModifiers{})
}

// PressKeyWithModifiers presses a key with modifier keys (Ctrl, Alt, Shift, Meta).
func (c *Client) PressKeyWithModifiers(ctx context.Context, targetID string, key string, mods KeyModifiers) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	// Get key code if available
	keyCode, hasKeyCode := keyCodeMap[key]
	modMask := mods.modifierBitmask()

	params := map[string]interface{}{
		"type":      "keyDown",
		"key":       key,
		"modifiers": modMask,
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

// Hover moves the mouse over an element specified by selector.
func (c *Client) Hover(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	x, y, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return err
	}

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

// Focus focuses on an element specified by selector.
func (c *Client) Focus(ctx context.Context, targetID string, selector string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Runtime.enable", nil)
	if err != nil {
		return fmt.Errorf("enabling Runtime domain: %w", err)
	}

	nodeID, err := c.resolveNodeID(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "DOM.focus", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return fmt.Errorf("focusing element: %w", err)
	}

	return nil
}

// Swipe performs a touch swipe gesture on an element.
func (c *Client) Swipe(ctx context.Context, targetID string, selector string, direction string) (*SwipeResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	cx, cy, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return nil, err
	}

	// Calculate swipe delta based on direction
	var dx, dy float64
	swipeDist := 100.0
	switch direction {
	case "left":
		dx, dy = -swipeDist, 0
	case "right":
		dx, dy = swipeDist, 0
	case "up":
		dx, dy = 0, -swipeDist
	case "down":
		dx, dy = 0, swipeDist
	default:
		return nil, fmt.Errorf("invalid direction: %s (use left, right, up, down)", direction)
	}

	// touchStart at center
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type": "touchStart",
		"touchPoints": []map[string]interface{}{
			{"x": cx, "y": cy},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dispatching touchStart: %w", err)
	}

	// Intermediate touchMove steps for smooth swipe
	steps := 5
	for i := 1; i <= steps; i++ {
		frac := float64(i) / float64(steps)
		_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
			"type": "touchMove",
			"touchPoints": []map[string]interface{}{
				{"x": cx + dx*frac, "y": cy + dy*frac},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("dispatching touchMove: %w", err)
		}
	}

	// touchEnd
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type":        "touchEnd",
		"touchPoints": []map[string]interface{}{},
	})
	if err != nil {
		return nil, fmt.Errorf("dispatching touchEnd: %w", err)
	}

	return &SwipeResult{
		Swiped:    true,
		Direction: direction,
		Selector:  selector,
	}, nil
}

// Pinch performs a two-finger pinch gesture on an element.
func (c *Client) Pinch(ctx context.Context, targetID string, selector string, direction string) (*PinchResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	cx, cy, err := c.resolveElementCenter(ctx, sessionID, selector)
	if err != nil {
		return nil, err
	}

	// Two-finger pinch: fingers start/end at different offsets
	var startOffset, endOffset float64
	switch direction {
	case "in":
		startOffset, endOffset = 50, 10 // fingers converge
	case "out":
		startOffset, endOffset = 10, 50 // fingers diverge
	default:
		return nil, fmt.Errorf("invalid direction: %s (use in, out)", direction)
	}

	// touchStart with two fingers
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type": "touchStart",
		"touchPoints": []map[string]interface{}{
			{"x": cx - startOffset, "y": cy},
			{"x": cx + startOffset, "y": cy},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dispatching touchStart: %w", err)
	}

	// Intermediate steps
	steps := 5
	for i := 1; i <= steps; i++ {
		frac := float64(i) / float64(steps)
		offset := startOffset + (endOffset-startOffset)*frac
		_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
			"type": "touchMove",
			"touchPoints": []map[string]interface{}{
				{"x": cx - offset, "y": cy},
				{"x": cx + offset, "y": cy},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("dispatching touchMove: %w", err)
		}
	}

	// touchEnd
	_, err = c.CallSession(ctx, sessionID, "Input.dispatchTouchEvent", map[string]interface{}{
		"type":        "touchEnd",
		"touchPoints": []map[string]interface{}{},
	})
	if err != nil {
		return nil, fmt.Errorf("dispatching touchEnd: %w", err)
	}

	return &PinchResult{
		Pinched:   true,
		Direction: direction,
		Selector:  selector,
	}, nil
}

// MouseMove moves the mouse to the specified coordinates without clicking.
func (c *Client) MouseMove(ctx context.Context, targetID string, x, y float64) (*MouseMoveResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	_, err = c.CallSession(ctx, sessionID, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return nil, fmt.Errorf("moving mouse: %w", err)
	}

	return &MouseMoveResult{X: x, Y: y}, nil
}

// SelectOption selects an option in a <select> element by value.
func (c *Client) SelectOption(ctx context.Context, targetID string, selector string, value string) error {
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

// UploadFile sets files for a file input element.
func (c *Client) UploadFile(ctx context.Context, targetID string, selector string, files []string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	nodeID, err := c.resolveNodeID(ctx, sessionID, selector)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "DOM.setFileInputFiles", map[string]interface{}{
		"nodeId": nodeID,
		"files":  files,
	})
	if err != nil {
		return fmt.Errorf("setting files: %w", err)
	}

	return nil
}

// DispatchEvent dispatches a custom event on an element.
func (c *Client) DispatchEvent(ctx context.Context, targetID string, selector string, eventType string) (*DispatchEventResult, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	jsExpr := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) {
				return {error: 'element not found'};
			}
			const event = new Event(%q, {bubbles: true, cancelable: true});
			el.dispatchEvent(event);
			return {dispatched: true};
		})()
	`, selector, eventType)

	result, err := c.CallSession(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    jsExpr,
		"returnByValue": true,
	})
	if err != nil {
		return nil, fmt.Errorf("dispatching event: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value struct {
				Error      string `json:"error"`
				Dispatched bool   `json:"dispatched"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if evalResp.Result.Value.Error != "" {
		return nil, fmt.Errorf("%s", evalResp.Result.Value.Error)
	}

	return &DispatchEventResult{
		Dispatched: true,
		EventType:  eventType,
		Selector:   selector,
	}, nil
}
