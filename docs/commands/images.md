# hubcap images

List all images on the current page with src, alt text, and dimensions.

## When to use

Use `images` to enumerate every image on the page. Useful for auditing image assets, checking for missing alt text, or verifying image dimensions.

## Usage

```
hubcap images
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| images | array | Array of image objects |
| images[].src | string | The image source URL |
| images[].alt | string | The alt text attribute |
| images[].width | number | The rendered width in pixels |
| images[].height | number | The rendered height in pixels |
| count | number | Total number of images found |

```json
{
  "images": [
    {
      "src": "https://example.com/logo.png",
      "alt": "Company Logo",
      "width": 200,
      "height": 50
    },
    {
      "src": "https://example.com/hero.jpg",
      "alt": "",
      "width": 1200,
      "height": 600
    }
  ],
  "count": 2
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all images:

```
hubcap images
```

Find images with missing alt text:

```
hubcap images | jq '[.images[] | select(.alt == "")]'
```

Generate a report of image sizes for a page by chaining with other tools:

```
hubcap goto "https://example.com"
hubcap images | jq -r '.images[] | "\(.width)x\(.height) \(.src)"'
```

## See also

- [links](links.md) - extract all links from the page
- [scripts](scripts.md) - list all script elements
