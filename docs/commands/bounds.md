# hubcap bounds -- Get element bounding box

## When to use

Get the bounding box of an element for coordinate calculations. Use with `clickat` to click at relative positions within or near an element.

## Usage

```
hubcap bounds <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| x | number | Left edge x-coordinate in pixels |
| y | number | Top edge y-coordinate in pixels |
| width | number | Width in pixels |
| height | number | Height in pixels |

```json
{"x":100,"y":200,"width":300,"height":50}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the bounding box of a header:

```
hubcap bounds 'header'
```

Get bounds and compute the center point for a click:

```
hubcap bounds '.canvas' | jq '{x: (.x + .width/2), y: (.y + .height/2)}'
```

## See also

- [clickat](clickat.md) - Click at specific coordinates
- [mouse](mouse.md) - Move mouse to coordinates
- [visible](visible.md) - Check if an element is visible
- [layout](layout.md) - Get element layout with children
