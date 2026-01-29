# hubcap url

Get the URL of the current page.

## When to use

Use `url` to retrieve just the current page URL. Use `info` if you need the URL along with the title and meta information in a single call. Use `waiturl` to block until the URL matches a specific pattern.

## Usage

```
hubcap url
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| url | string | The URL of the current page |

```json
{
  "url": "https://example.com"
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Print the current URL:

```
hubcap url
```

Extract just the URL string:

```
hubcap url | jq -r '.url'
```

Navigate and verify the final URL after redirects:

```
hubcap goto "https://example.com/redirect"
hubcap waitload
FINAL=$(hubcap url | jq -r '.url')
echo "Landed on: $FINAL"
```

## See also

- [title](title.md) - get the current page title
- [info](info.md) - get combined page information (title, URL, meta)
- [waiturl](waiturl.md) - wait for the URL to match a pattern
