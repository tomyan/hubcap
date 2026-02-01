# hubcap rightclick -- Right-click an element

## When to use

Right-click to open context menus. Use `click` for a standard left-click.

## Usage

```
hubcap rightclick <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to right-click |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| clicked | boolean | Whether the right-click succeeded |
| selector | string | The selector that was used |

```json
{"clicked":true,"selector":"#canvas"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Right-click to open a context menu:

```
hubcap rightclick '#canvas'
```

Right-click an element, then click a context menu item:

```
hubcap rightclick '.file-row' && hubcap click '.context-menu .delete'
```

## See also

- [click](click.md) - Single-click an element
- [dblclick](dblclick.md) - Double-click an element
- [hover](hover.md) - Hover over an element
