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

Returns an array of stylesheet info objects.

| Field | Type | Description |
|-------|------|-------------|
| [].styleSheetId | string | Identifier for the stylesheet |
| [].sourceURL | string | URL of the stylesheet, empty for inline styles |
| [].title | string | Title of the stylesheet |
| [].disabled | boolean | Whether the stylesheet is disabled |
| [].isInline | boolean | Whether the stylesheet is an inline `<style>` block |

```json
[
  {
    "styleSheetId": "1",
    "sourceURL": "https://example.com/styles.css",
    "title": "",
    "disabled": false,
    "isInline": false
  },
  {
    "styleSheetId": "2",
    "sourceURL": "",
    "title": "",
    "disabled": false,
    "isInline": true
  }
]
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
hubcap stylesheets | jq '{external: [.[] | select(.isInline | not)] | length, inline: [.[] | select(.isInline)] | length}'
```

List all external stylesheet URLs for a page:

```
hubcap goto "https://example.com"
hubcap stylesheets | jq -r '[.[] | select(.sourceURL != "") | .sourceURL] | .[]'
```

## See also

- [csscoverage](csscoverage.md) - get CSS rule coverage
- [styles](styles.md) - get computed styles for an element
- [computed](computed.md) - get computed style properties
