package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// formatRemoteObject returns a text representation of a RemoteObject.
func formatRemoteObject(obj *RemoteObject) string {
	if obj.UnserializableValue != "" {
		return obj.UnserializableValue
	}
	switch obj.Type {
	case "string":
		if s, ok := obj.Value.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", obj.Value)
	case "number", "boolean":
		return fmt.Sprintf("%v", obj.Value)
	case "undefined":
		return "undefined"
	case "symbol":
		return obj.Description
	case "function":
		if obj.Description != "" {
			return obj.Description
		}
		return "function"
	case "object":
		if obj.Subtype == "null" {
			return "null"
		}
		return formatObjectPreview(obj)
	default:
		if obj.Value != nil {
			return fmt.Sprintf("%v", obj.Value)
		}
		return obj.Description
	}
}

// formatObjectPreview renders a one-line preview of an object.
func formatObjectPreview(obj *RemoteObject) string {
	if obj.Preview != nil {
		return formatPreview(obj.Preview)
	}
	if obj.Description != "" {
		return obj.Description
	}
	if obj.ClassName != "" {
		return obj.ClassName + " {…}"
	}
	return "Object"
}

// formatPreview renders an ObjectPreview as a one-line string.
func formatPreview(p *ObjectPreview) string {
	if p.Subtype == "array" {
		return formatArrayPreview(p)
	}
	if p.Subtype == "date" || p.Subtype == "regexp" {
		return p.Description
	}

	var parts []string
	for _, prop := range p.Properties {
		parts = append(parts, fmt.Sprintf("%s: %s", prop.Name, prop.Value))
	}
	inner := strings.Join(parts, ", ")
	if p.Overflow {
		inner += ", …"
	}

	prefix := ""
	if p.Description != "" && p.Description != "Object" {
		prefix = p.Description + " "
	}
	return prefix + "{" + inner + "}"
}

// formatArrayPreview renders an array preview.
func formatArrayPreview(p *ObjectPreview) string {
	var parts []string
	for _, prop := range p.Properties {
		// Array previews include numeric indices and "length"
		if prop.Name == "length" {
			continue
		}
		parts = append(parts, prop.Value)
	}
	inner := strings.Join(parts, ", ")
	if p.Overflow {
		inner += ", …"
	}
	return "[" + inner + "]"
}

// FormatRemoteObject is the exported version for use outside the package.
func FormatRemoteObject(obj *RemoteObject) string {
	return formatRemoteObject(obj)
}

// GetProperties fetches the properties of a remote object.
func (c *Client) GetProperties(ctx context.Context, sessionID, objectID string, ownOnly bool) ([]PropertyDescriptor, error) {
	params := map[string]interface{}{
		"objectId":               objectID,
		"ownProperties":          ownOnly,
		"generatePreview":        true,
		"accessorPropertiesOnly": false,
	}

	result, err := c.CallSession(ctx, sessionID, "Runtime.getProperties", params)
	if err != nil {
		return nil, fmt.Errorf("getProperties: %w", err)
	}

	var resp struct {
		Result             []PropertyDescriptor `json:"result"`
		InternalProperties []PropertyDescriptor `json:"internalProperties"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parsing properties: %w", err)
	}

	// Append internal properties (like [[Prototype]]) after own properties.
	all := resp.Result
	for _, ip := range resp.InternalProperties {
		all = append(all, ip)
	}
	return all, nil
}

// ReleaseObjectGroup releases all remote objects in a group.
func (c *Client) ReleaseObjectGroup(ctx context.Context, sessionID, group string) error {
	_, err := c.CallSession(ctx, sessionID, "Runtime.releaseObjectGroup", map[string]interface{}{
		"objectGroup": group,
	})
	return err
}

// ReleaseObject releases a single remote object.
func (c *Client) ReleaseObject(ctx context.Context, sessionID, objectID string) error {
	_, err := c.CallSession(ctx, sessionID, "Runtime.releaseObject", map[string]interface{}{
		"objectId": objectID,
	})
	return err
}
