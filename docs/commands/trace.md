# hubcap trace - Capture a Chrome performance trace

## When to use

Capture a Chrome performance trace for CPU profiling and runtime analysis. Open the output file in Chrome DevTools Performance panel to visualize flame charts, paint events, and layout shifts. Use `metrics` for a quick performance check without generating a full trace.

## Usage

```
hubcap trace --output <file> [--duration <duration>]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `""` | Output file path (required) |
| `--duration` | duration | `1s` | Trace duration |

## Output

| Field | Type | Description |
|-------|------|-------------|
| `file` | string | Path to the written trace file |
| `size` | int | File size in bytes |

```json
{"file":"trace.json","size":54321}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Missing --output flag | 1 | `usage: hubcap trace --duration <d> --output <file>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Cannot write to file | 1 | `error: writing file: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Capture a 1-second trace (default duration):

```bash
hubcap trace --output trace.json
```

Capture a 5-second trace:

```bash
hubcap trace --output trace.json --duration 5s
```

Navigate to a page and trace its load performance:

```bash
hubcap goto "https://example.com" && hubcap trace --output load-trace.json --duration 3s
```

Capture a trace with a timestamped filename and report the size:

```bash
hubcap trace --output "trace-$(date +%s).json" --duration 2s | jq '"Trace size: \(.size) bytes"'
```

## See also

- [heapsnapshot](heapsnapshot.md) - Capture a V8 heap snapshot
- [metrics](metrics.md) - Get page performance metrics
- [coverage](coverage.md) - Collect code coverage data
