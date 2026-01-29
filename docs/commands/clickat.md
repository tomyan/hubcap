# hubcap clickat

Click at specific page coordinates.

## When to use

Click at exact viewport coordinates when you know the pixel position. Use `click` when you have a CSS selector instead. Use `bounds` to get an element's coordinates first.

## Usage

```
hubcap clickat <x> <y>
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
| `clicked` | boolean | Whether the click succeeded |
| `x` | number | The X coordinate that was clicked |
| `y` | number | The Y coordinate that was clicked |

```json
{"clicked":true,"x":100,"y":200}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Invalid x coordinate | 1 | `error: invalid x coordinate: ...` |
| Invalid y coordinate | 1 | `error: invalid y coordinate: ...` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Click at the center of a 1280x720 viewport:

```
hubcap clickat 640 360
```

Click near the top-left corner:

```
hubcap clickat 10 10
```

Get an element's position with `bounds`, then click it:

```
hubcap bounds '#submit' | jq '{x: .x, y: .y}' | xargs -I {} sh -c 'hubcap clickat $(echo {} | jq -r ".x") $(echo {} | jq -r ".y")'
```

## See also

- [click](click.md) - Click an element by CSS selector
- [mouse](mouse.md) - Move mouse to coordinates without clicking
- [bounds](bounds.md) - Get element bounding box coordinates
- [hover](hover.md) - Hover over an element by selector
