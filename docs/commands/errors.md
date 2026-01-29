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
| `message` | string | Exception message |
| `stackTrace` | string | Full stack trace |

```json
{"message":"TypeError: Cannot read properties of undefined (reading 'map')","stackTrace":"TypeError: Cannot read properties of undefined (reading 'map')\n    at render (app.js:42:15)\n    at update (app.js:30:5)"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not reachable | 2 | `error: cannot connect to Chrome` |
| Duration parse failure | 1 | `error: invalid duration "<value>"` |

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
hubcap navigate "https://example.com" && hubcap errors --duration 5s | jq -r '"[\(.message)]\n\(.stackTrace)\n"'
```

## See also

- [console](console.md) - Capture all browser console messages
- [eval](eval.md) - Evaluate JavaScript in the browser
- [network](network.md) - Stream network requests and responses
