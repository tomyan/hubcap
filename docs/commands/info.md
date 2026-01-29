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
| readyState | string | Document ready state (`loading`, `interactive`, `complete`) |
| characterSet | string | Character encoding (e.g. `UTF-8`) |
| contentType | string | Document content type (e.g. `text/html`) |

```json
{
  "title": "Example Domain",
  "url": "https://example.com",
  "readyState": "complete",
  "characterSet": "UTF-8",
  "contentType": "text/html"
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
hubcap goto "https://example.com" && hubcap info | jq -e '.readyState == "complete"'
```

## See also

- [title](title.md) - get just the page title
- [url](url.md) - get just the page URL
- [meta](meta.md) - get meta tags from the page
- [source](source.md) - get the full HTML source
