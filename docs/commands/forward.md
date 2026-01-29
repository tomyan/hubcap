# hubcap forward

Navigate forward one entry in the browser's session history.

## When to use

Use `forward` to go to the next page in the tab's history, equivalent to pressing the browser's forward button. This only works after a `back` navigation. Use `goto` to navigate to a specific URL instead.

## Usage

```
hubcap forward
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

Go forward one page:

```
hubcap forward
```

Go back and then forward, confirming the URL is restored:

```
ORIGINAL=$(hubcap url | jq -r '.url')
hubcap back
hubcap forward
hubcap url | jq -r '.url'  # should match $ORIGINAL
```

Chain forward with a wait for network idle:

```
hubcap forward && hubcap waitidle
```

## See also

- [back](back.md) - navigate back in browser history
- [goto](goto.md) - navigate to a specific URL
- [reload](reload.md) - reload the current page
