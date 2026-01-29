# hubcap meta

Get all meta tags from the current page.

## When to use

Use `meta` to retrieve every `<meta>` tag on the page, including name, property, content, charset, and http-equiv attributes. Use `info` for a combined overview that includes some meta information alongside title and URL.

## Usage

```
hubcap meta
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| tags | array | Array of meta tag objects |
| tags[].name | string | The `name` attribute value |
| tags[].property | string | The `property` attribute value (Open Graph, etc.) |
| tags[].content | string | The `content` attribute value |
| tags[].charset | string | The `charset` attribute value |
| tags[].httpEquiv | string | The `http-equiv` attribute value |

```json
{
  "tags": [
    {
      "name": "description",
      "content": "An example page for testing."
    },
    {
      "property": "og:title",
      "content": "Example Domain"
    }
  ]
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get all meta tags:

```
hubcap meta
```

Extract the description meta tag:

```
hubcap meta | jq -r '.tags[] | select(.name=="description") | .content'
```

Audit Open Graph tags by chaining with `jq`:

```
hubcap goto "https://example.com" && hubcap meta | jq '[.tags[] | select(.property | startswith("og:"))]'
```

## See also

- [info](info.md) - get combined page information including meta
- [links](links.md) - extract all links from the page
- [source](source.md) - get the full HTML source
