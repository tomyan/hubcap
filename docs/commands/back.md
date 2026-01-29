# hubcap back

Navigate back one entry in the browser's session history.

## When to use

Use `back` to go to the previous page in the tab's history, equivalent to pressing the browser's back button. Use `goto` to navigate to a specific URL instead, or `forward` to go the other direction.

## Usage

```
hubcap back
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| success | boolean | Whether the navigation completed |

```json
{
  "success": true
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Go back one page:

```
hubcap back
```

Navigate to a page, go back, then verify the URL changed:

```
hubcap goto "https://example.com/page2"
hubcap back
hubcap url | jq -r '.url'
```

Go back and wait for the page to finish loading:

```
hubcap back && hubcap waitload
```

## See also

- [forward](forward.md) - navigate forward in browser history
- [goto](goto.md) - navigate to a specific URL
- [reload](reload.md) - reload the current page
