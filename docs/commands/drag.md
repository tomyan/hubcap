# hubcap drag

Drag one element onto another.

## When to use

Use `drag` to simulate a drag-and-drop interaction between two elements. Both the source and destination selectors must match visible elements. Use `mouse` for more granular pointer control when the standard drag gesture is not sufficient.

## Usage

```
hubcap drag <source-selector> <dest-selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `source-selector` | string | Yes | CSS selector of the element to drag |
| `dest-selector` | string | Yes | CSS selector of the drop target |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `dragged` | boolean | Whether the drag succeeded |
| `source` | string | The selector of the dragged element |
| `dest` | string | The selector of the drop target |

```json
{"dragged":true,"source":"#item-1","dest":"#dropzone"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Drag a list item to a drop zone:

```
hubcap drag '#item-1' '#dropzone'
```

Reorder items in a sortable list:

```
hubcap drag '.sortable:first-child' '.sortable:last-child'
```

Drag an item then verify the drop zone updated:

```
hubcap drag '#card' '#done-column' && hubcap text '#done-column'
```

## See also

- [click](click.md) - Click an element
- [mouse](mouse.md) - Move the mouse to coordinates
- [hover](hover.md) - Hover over an element
