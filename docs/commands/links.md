# hubcap links

Extract all links from the current page.

## When to use

Use `links` to get every anchor element's href and text content. Useful for crawling, sitemap validation, or checking for broken links.

## Usage

```
hubcap links
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| links | array | Array of link objects |
| links[].href | string | The `href` attribute value |
| links[].text | string | The visible text content of the link |

```json
{
  "links": [
    {
      "href": "https://example.com/about",
      "text": "About Us"
    },
    {
      "href": "https://example.com/contact",
      "text": "Contact"
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

List all links on the page:

```
hubcap links
```

Extract only external links:

```
hubcap links | jq '[.links[] | select(.href | startswith("http")) | select(.href | contains("example.com") | not)]'
```

Check each link for broken URLs by piping into `curl`:

```
hubcap links | jq -r '.links[].href' | while read -r url; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$url")
  echo "$STATUS $url"
done
```

## See also

- [meta](meta.md) - get meta tags from the page
- [images](images.md) - list all images on the page
- [scripts](scripts.md) - list all script elements
