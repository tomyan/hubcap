# hubcap coverage

Get JavaScript code coverage data for the current page.

## When to use

Use `coverage` to identify unused JavaScript code. For the most accurate results, run this after performing page interactions so that event handlers and lazy code paths are exercised. Use `csscoverage` for CSS rule coverage instead.

## Usage

```
hubcap coverage
```

## Arguments

None.

## Flags

None.

## Output

Returns JavaScript code coverage data for each script.

| Field | Type | Description |
|-------|------|-------------|
| [].scriptId | string | Identifier of the script |
| [].url | string | URL of the script |
| [].functions | array | Array of function coverage entries |
| [].functions[].functionName | string | Name of the function |
| [].functions[].ranges | array | Array of coverage ranges with start/end offsets and count |

```json
[
  {
    "scriptId": "42",
    "url": "https://example.com/app.js",
    "functions": [
      {
        "functionName": "init",
        "ranges": [
          {"startOffset": 0, "endOffset": 120, "count": 1}
        ]
      },
      {
        "functionName": "unusedHelper",
        "ranges": [
          {"startOffset": 121, "endOffset": 200, "count": 0}
        ]
      }
    ]
  }
]
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get JavaScript coverage:

```
hubcap coverage
```

Find functions that were never called:

```
hubcap coverage | jq '[.[] | .functions[] | select(.ranges[0].count == 0) | .functionName]'
```

Interact with the page first, then measure coverage:

```
hubcap goto "https://example.com"
hubcap click "#menu-toggle"
hubcap click "#submit"
hubcap coverage | jq '[.[] | {url, unused: [.functions[] | select(.ranges[0].count == 0)] | length}]'
```

## See also

- [csscoverage](csscoverage.md) - get CSS rule coverage
- [scripts](scripts.md) - list all script elements
- [metrics](metrics.md) - get performance metrics
