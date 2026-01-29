# hubcap scrollto -- Scroll an element into view

## When to use

Scroll an element into view. Use before `click` if the element is off-screen. Use `scroll` for pixel-based scrolling.

## Usage

```
hubcap scrollto <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to scroll into view |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| scrolled | boolean | Whether the scroll succeeded |
| selector | string | The selector that was used |

```json
{"scrolled":true,"selector":"#footer"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Scroll to the footer:

```
hubcap scrollto '#footer'
```

Scroll an element into view, then click it:

```
hubcap scrollto '.load-more' && hubcap click '.load-more'
```

## See also

- [scroll](scroll.md) - Scroll by pixel amount
- [scrolltop](scrolltop.md) - Scroll to the top of the page
- [scrollbottom](scrollbottom.md) - Scroll to the bottom of the page
- [click](click.md) - Single-click an element
- [bounds](bounds.md) - Get element bounding box
