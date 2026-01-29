# hubcap mouse

Move the mouse to specific viewport coordinates without clicking.

## When to use

Move the mouse pointer to exact coordinates without triggering a click. Use `hover` to move to an element by CSS selector. Use `clickat` to move and click in one step.

## Usage

```
hubcap mouse <x> <y>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `x` | float | Yes | X coordinate in pixels from the left edge of the viewport |
| `y` | float | Yes | Y coordinate in pixels from the top edge of the viewport |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `x` | number | The X coordinate the mouse moved to |
| `y` | number | The Y coordinate the mouse moved to |

```json
{"x":100,"y":200}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Invalid coordinate (non-numeric or out of range) | 1 | `invalid coordinate` |
| Chrome not connected | 2 | `chrome: not connected` |
| Operation timed out | 3 | `timeout` |

## Examples

Move the mouse to the center of the viewport:

```
hubcap mouse 640 360
```

Move the mouse to trigger a hover effect at a specific point:

```
hubcap mouse 250 75
```

Move to an element's position using `bounds`, then take a screenshot of the hover state:

```
pos=$(hubcap bounds '.tooltip-trigger' | jq -r '"\(.x) \(.y)"') && hubcap mouse $pos && hubcap screenshot hover.png
```

## See also

- [clickat](clickat.md) - Click at specific coordinates
- [hover](hover.md) - Hover over an element by CSS selector
- [drag](drag.md) - Drag from one element to another
- [bounds](bounds.md) - Get element bounding box coordinates
