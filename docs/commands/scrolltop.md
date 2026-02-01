# hubcap scrolltop

Scroll to the top of the page.

## When to use

Use `scrolltop` to scroll the page to the very top. Use `scrollto` to scroll to a specific element, `scrollbottom` to go to the end, or `scroll` for relative pixel-based scrolling.

## Usage

```
hubcap scrolltop
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| scrolled | boolean | Whether the scroll action completed |

```json
{
  "scrolled": true
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Scroll to the top:

```
hubcap scrolltop
```

Scroll to the bottom to trigger lazy loading, then back to the top:

```
hubcap scrollbottom && hubcap waitidle && hubcap scrolltop
```

Take a screenshot from the top of the page:

```
hubcap scrolltop && hubcap screenshot --output top.png
```

## See also

- [scrollbottom](scrollbottom.md) - scroll to the bottom of the page
- [scrollto](scrollto.md) - scroll to a specific element
- [scroll](scroll.md) - scroll by a relative amount
