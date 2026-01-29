# hubcap scrollbottom

Scroll to the bottom of the page.

## When to use

Use `scrollbottom` to scroll the page all the way to the end. Useful for triggering lazy-loaded content or infinite scroll. Use `scrollto` to scroll to a specific element, or `scrolltop` to return to the top.

## Usage

```
hubcap scrollbottom
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

Scroll to the bottom:

```
hubcap scrollbottom
```

Trigger infinite scroll by scrolling to the bottom and waiting for new content:

```
hubcap scrollbottom
hubcap waitidle
hubcap scrollbottom
hubcap waitidle
```

Scroll to the bottom and count the total number of items loaded:

```
hubcap scrollbottom && hubcap waitidle
hubcap count ".item"
```

## See also

- [scrolltop](scrolltop.md) - scroll to the top of the page
- [scrollto](scrollto.md) - scroll to a specific element
- [scroll](scroll.md) - scroll by a relative amount
