# hubcap title

Get the title of the current page.

## When to use

Use `title` to retrieve just the document title of the active page. Use `info` if you need the title, URL, and meta information in a single call.

## Usage

```
hubcap title
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| title | string | The document title of the current page |

```json
{
  "title": "Example Domain"
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Print the page title:

```
hubcap title
```

Extract just the title string:

```
hubcap title | jq -r '.title'
```

Navigate to a page, then assert the title matches an expected value:

```
hubcap goto "https://example.com"
hubcap title | jq -e '.title == "Example Domain"'
```

## See also

- [url](url.md) - get the current page URL
- [info](info.md) - get combined page information (title, URL, meta)
