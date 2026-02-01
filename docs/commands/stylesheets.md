# hubcap stylesheets

List all stylesheets loaded on the current page.

## When to use

Use `stylesheets` to enumerate every stylesheet attached to the page, including external files and inline styles. Useful for auditing CSS assets, detecting unused stylesheets, or verifying that expected styles are loaded.

## Usage

```
hubcap stylesheets
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| stylesheets | array | Array of stylesheet info objects |
| stylesheets[].styleSheetId | string | Identifier for the stylesheet |
| stylesheets[].sourceURL | string | URL of the stylesheet, empty for inline styles |
| stylesheets[].title | string | Title of the stylesheet |
| stylesheets[].disabled | boolean | Whether the stylesheet is disabled |
| stylesheets[].isInline | boolean | Whether the stylesheet is an inline `<style>` block |
| stylesheets[].length | number | Number of CSS rules in the stylesheet |

```json
{
  "stylesheets": [
    {
      "styleSheetId": "0",
      "sourceURL": "https://example.com/styles.css",
      "title": "",
      "disabled": false,
      "isInline": false,
      "length": 42
    },
    {
      "styleSheetId": "1",
      "sourceURL": "",
      "title": "",
      "disabled": false,
      "isInline": true,
      "length": 5
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

List all stylesheets:

```
hubcap stylesheets
```

Count external vs inline stylesheets:

```
hubcap stylesheets | jq '{external: [.stylesheets[] | select(.isInline | not)] | length, inline: [.stylesheets[] | select(.isInline)] | length}'
```

List all external stylesheet URLs for a page:

```
hubcap goto "https://example.com" && hubcap stylesheets | jq -r '[.stylesheets[] | select(.sourceURL != "") | .sourceURL] | .[]'
```

## See also

- [csscoverage](csscoverage.md) - get CSS rule coverage
- [styles](styles.md) - get computed styles for an element
- [computed](computed.md) - get computed style properties
