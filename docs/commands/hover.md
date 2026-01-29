# hubcap hover -- Hover over an element

## When to use

Hover over an element to trigger `:hover` styles, tooltips, or dropdown menus. Use `mouse` to move to specific coordinates instead of a selector.

## Usage

```
hubcap hover <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to hover over |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| hovered | boolean | Whether the hover succeeded |
| selector | string | The selector that was used |

```json
{"hovered":true,"selector":".menu-trigger"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Hover over a menu trigger to open a dropdown:

```
hubcap hover '.menu-trigger'
```

Hover to reveal a tooltip, then read its text:

```
hubcap hover '.info-icon' && hubcap text '.tooltip'
```

## See also

- [click](click.md) - Single-click an element
- [mouse](mouse.md) - Move the mouse to coordinates
- [styles](styles.md) - Get computed CSS styles
