# hubcap console - Capture browser console messages

## When to use

Capture console messages from the browser in real time, including log, warn, error, info, and debug levels. Use `errors` if you only need JavaScript exceptions with stack traces.

## Usage

```
hubcap console [--duration <duration>]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--duration` | duration | `0` | How long to capture; 0 = until interrupted |

## Output

NDJSON stream written to stdout. Each line is a JSON object representing a console message.

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Console level: `"log"`, `"warn"`, `"error"`, `"info"`, or `"debug"` |
| `text` | string | The console message text |

```json
{"type":"log","text":"Page loaded successfully"}
```

```json
{"type":"warn","text":"Deprecated API usage detected"}
```

```json
{"type":"error","text":"Failed to fetch resource"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not reachable | 2 | `error: cannot connect to Chrome` |
| Duration parse failure | 1 | `error: invalid duration "<value>"` |

## Examples

Stream all console messages until Ctrl-C:

```bash
hubcap console
```

Capture console output for 15 seconds:

```bash
hubcap console --duration 15s
```

Filter for errors and warnings only:

```bash
hubcap console | jq 'select(.type == "error" or .type == "warn")'
```

Navigate to a page and capture its console output:

```bash
hubcap navigate "https://example.com" && hubcap console --duration 10s | jq -r '.text'
```

## See also

- [errors](errors.md) - Capture JavaScript exceptions with stack traces
- [eval](eval.md) - Evaluate JavaScript in the browser
- [network](network.md) - Stream network requests and responses
