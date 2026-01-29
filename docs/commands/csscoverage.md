# hubcap csscoverage

Get CSS rule coverage data for the current page.

## When to use

Use `csscoverage` to identify unused CSS rules. For the most accurate results, run this after performing page interactions that may trigger hover states, media queries, or dynamic class changes. Use `coverage` for JavaScript code coverage instead.

## Usage

```
hubcap csscoverage
```

## Arguments

None.

## Flags

None.

## Output

Returns CSS rule coverage data for each stylesheet.

| Field | Type | Description |
|-------|------|-------------|
| [].styleSheetId | string | Identifier of the stylesheet |
| [].url | string | URL of the stylesheet |
| [].ranges | array | Array of used byte ranges |
| [].ranges[].startOffset | number | Start byte offset of a used range |
| [].ranges[].endOffset | number | End byte offset of a used range |
| [].text | string | Full text of the stylesheet |

```json
[
  {
    "styleSheetId": "1",
    "url": "https://example.com/styles.css",
    "ranges": [
      {"startOffset": 0, "endOffset": 500},
      {"startOffset": 800, "endOffset": 1200}
    ],
    "text": "body { margin: 0; } .header { ... } .unused { ... }"
  }
]
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get CSS coverage:

```
hubcap csscoverage
```

Calculate the percentage of CSS used per stylesheet:

```
hubcap csscoverage | jq '[.[] | {url, total: (.text | length), used: ([.ranges[] | .endOffset - .startOffset] | add), pct: (([.ranges[] | .endOffset - .startOffset] | add) / (.text | length) * 100 | round)}]'
```

Navigate and interact before measuring coverage:

```
hubcap goto --wait "https://example.com" && hubcap hover ".dropdown-trigger" && hubcap click "#tab-2" && hubcap csscoverage > css-coverage.json
```

## See also

- [coverage](coverage.md) - get JavaScript code coverage
- [stylesheets](stylesheets.md) - list all stylesheets
