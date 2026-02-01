# hubcap scroll

Scroll the page by a relative pixel amount.

## When to use

Scroll the page by a given number of pixels on the x and y axes. Use `scrollto` to scroll a specific element into view. Use `scrolltop` or `scrollbottom` to jump to the top or bottom of the page.

## Usage

```
hubcap scroll <x> <y>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `x` | int | Yes | Horizontal scroll distance in pixels (positive = right, negative = left) |
| `y` | int | Yes | Vertical scroll distance in pixels (positive = down, negative = up) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `scrolled` | boolean | Whether the scroll was performed |
| `x` | number | Horizontal distance scrolled |
| `y` | number | Vertical distance scrolled |

```json
{"scrolled":true,"x":0,"y":500}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Invalid x value | 1 | `invalid x value: <val>` |
| Invalid y value | 1 | `invalid y value: <val>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Scroll down 500 pixels:

```
hubcap scroll 0 500
```

Scroll right 200 pixels:

```
hubcap scroll 200 0
```

Scroll diagonally:

```
hubcap scroll 100 300
```

Scroll down and take a screenshot of the newly visible content:

```
hubcap scroll 0 800 && hubcap screenshot --output after-scroll.png
```

## See also

- [scrollto](scrollto.md) - Scroll a specific element into view
- [scrolltop](scrolltop.md) - Scroll to the top of the page
- [scrollbottom](scrollbottom.md) - Scroll to the bottom of the page
