# hubcap info

Get combined page information including title, URL, and meta details in a single call.

## When to use

Use `info` to retrieve the page title, URL, and meta information together. Prefer this over making separate `title` and `url` calls when you need multiple pieces of page information.

## Usage

```
hubcap info
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| title | string | The document title |
| url | string | The current page URL |
| securityOrigin | string | The security origin of the page |
| secureContextType | string | Whether the page is a secure context |
| frameId | string | The main frame identifier |

```json
{
  "title": "Example Domain",
  "url": "https://example.com",
  "securityOrigin": "https://example.com",
  "secureContextType": "Secure",
  "frameId": "A1B2C3D4E5F6"
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get page info:

```
hubcap info
```

Print just the title and URL:

```
hubcap info | jq '{title, url}'
```

After navigation, confirm the page is on a secure context:

```
hubcap goto "https://example.com"
hubcap info | jq -e '.secureContextType == "Secure"'
```

## See also

- [title](title.md) - get just the page title
- [url](url.md) - get just the page URL
- [meta](meta.md) - get meta tags from the page
- [source](source.md) - get the full HTML source
