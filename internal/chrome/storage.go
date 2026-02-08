package chrome

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetCookies returns all cookies for the page.
func (c *Client) GetCookies(ctx context.Context, targetID string) ([]Cookie, error) {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Get cookies via Network domain
	result, err := c.CallSession(ctx, sessionID, "Network.getCookies", nil)
	if err != nil {
		return nil, fmt.Errorf("getting cookies: %w", err)
	}

	var resp struct {
		Cookies []Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing cookies response: %w", err)
	}

	return resp.Cookies, nil
}

// SetCookie sets a cookie for the page.
func (c *Client) SetCookie(ctx context.Context, targetID string, cookie Cookie) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"name":  cookie.Name,
		"value": cookie.Value,
	}
	if cookie.Domain != "" {
		params["domain"] = cookie.Domain
	}
	if cookie.Path != "" {
		params["path"] = cookie.Path
	}
	if cookie.Expires > 0 {
		params["expires"] = cookie.Expires
	}
	if cookie.HTTPOnly {
		params["httpOnly"] = cookie.HTTPOnly
	}
	if cookie.Secure {
		params["secure"] = cookie.Secure
	}
	if cookie.SameSite != "" {
		params["sameSite"] = cookie.SameSite
	}

	_, err = c.CallSession(ctx, sessionID, "Network.setCookie", params)
	if err != nil {
		return fmt.Errorf("setting cookie: %w", err)
	}

	return nil
}

// DeleteCookie deletes a cookie by name and domain.
func (c *Client) DeleteCookie(ctx context.Context, targetID string, name, domain string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"name": name,
	}
	if domain != "" {
		params["domain"] = domain
	}

	_, err = c.CallSession(ctx, sessionID, "Network.deleteCookies", params)
	if err != nil {
		return fmt.Errorf("deleting cookie: %w", err)
	}

	return nil
}

// ClearCookies clears all cookies.
func (c *Client) ClearCookies(ctx context.Context, targetID string) error {
	sessionID, err := c.attachToTarget(ctx, targetID)
	if err != nil {
		return err
	}

	_, err = c.CallSession(ctx, sessionID, "Network.clearBrowserCookies", nil)
	if err != nil {
		return fmt.Errorf("clearing cookies: %w", err)
	}

	return nil
}

// GetLocalStorage gets a value from localStorage.
func (c *Client) GetLocalStorage(ctx context.Context, targetID string, key string) (string, error) {
	js := fmt.Sprintf(`localStorage.getItem(%q)`, key)
	result, err := c.Eval(ctx, targetID, js)
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

// SetLocalStorage sets a value in localStorage.
func (c *Client) SetLocalStorage(ctx context.Context, targetID string, key, value string) error {
	js := fmt.Sprintf(`localStorage.setItem(%q, %q); true`, key, value)
	_, err := c.Eval(ctx, targetID, js)
	return err
}

// ClearLocalStorage clears all localStorage.
func (c *Client) ClearLocalStorage(ctx context.Context, targetID string) error {
	_, err := c.Eval(ctx, targetID, `localStorage.clear(); true`)
	return err
}

// GetSessionStorage gets a value from sessionStorage.
func (c *Client) GetSessionStorage(ctx context.Context, targetID string, key string) (string, error) {
	js := fmt.Sprintf(`sessionStorage.getItem(%q)`, key)
	result, err := c.Eval(ctx, targetID, js)
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

// SetSessionStorage sets a value in sessionStorage.
func (c *Client) SetSessionStorage(ctx context.Context, targetID string, key, value string) error {
	js := fmt.Sprintf(`sessionStorage.setItem(%q, %q); true`, key, value)
	_, err := c.Eval(ctx, targetID, js)
	return err
}

// ClearSessionStorage clears all sessionStorage.
func (c *Client) ClearSessionStorage(ctx context.Context, targetID string) error {
	_, err := c.Eval(ctx, targetID, `sessionStorage.clear(); true`)
	return err
}
