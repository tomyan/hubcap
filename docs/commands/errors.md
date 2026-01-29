# hubcap errors - Capture JavaScript exceptions with stack traces

## When to use

Capture JavaScript exceptions with full stack traces as they occur. Use `console` if you need all console messages including non-error output like log, warn, info, and debug.

## Usage

```
hubcap errors [--duration <duration>]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--duration` | duration | `0` | How long to capture; 0 = until interrupted |

## Output

NDJSON stream written to stdout. Each line is a JSON object representing a JavaScript exception.

| Field | Type | Description |
|-------|------|-------------|
| `text` | string | Exception text |
| `lineNumber` | number | Line number of the exception |
| `columnNumber` | number | Column number of the exception |
| `url` | string | URL where the exception occurred |

```json
{"text":"TypeError: Cannot read properties of undefined (reading 'map')","lineNumber":42,"columnNumber":15,"url":"https://example.com/app.js"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Stream all JavaScript exceptions until Ctrl-C:

```bash
hubcap errors
```

Capture exceptions for 30 seconds:

```bash
hubcap errors --duration 30s
```

Count exceptions by message:

```bash
hubcap errors --duration 60s | jq -r '.message' | sort | uniq -c | sort -rn
```

Navigate to a page and check for errors:

```bash
hubcap goto --wait "https://example.com" && hubcap errors --duration 5s | jq -r '"[\(.text)]\n\(.url):\(.lineNumber)\n"'
```

## See also

- [console](console.md) - Capture all browser console messages
- [eval](eval.md) - Evaluate JavaScript in the browser
- [network](network.md) - Stream network requests and responses
