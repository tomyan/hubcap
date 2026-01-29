# hubcap heapsnapshot - Capture a V8 heap snapshot

## When to use

Capture a V8 heap snapshot for memory analysis. Open the output file in Chrome DevTools Memory panel to inspect object allocations and track memory leaks. Use `metrics` for a quick heap size check without generating a full snapshot.

## Usage

```
hubcap heapsnapshot --output <file>
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `""` | Output file path (required) |

## Output

| Field | Type | Description |
|-------|------|-------------|
| `file` | string | Path to the written snapshot file |
| `size` | int | File size in bytes |

```json
{"file":"heap.json","size":12345}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not reachable | 2 | `error: cannot connect to Chrome` |
| Missing --output flag | 1 | `error: --output flag is required` |
| Cannot write to file | 1 | `error: cannot write to "<path>"` |
| Timeout during capture | 3 | `error: timeout` |

## Examples

Capture a heap snapshot:

```bash
hubcap heapsnapshot --output heap.json
```

Capture a snapshot and report its size:

```bash
hubcap heapsnapshot --output heap.json | jq '.size'
```

Compare heap size before and after an action:

```bash
hubcap heapsnapshot --output before.json
hubcap navigate "https://example.com/heavy-page"
hubcap heapsnapshot --output after.json
```

Take a snapshot and check quick metrics in one pipeline:

```bash
hubcap heapsnapshot --output "heap-$(date +%s).json" && hubcap metrics | jq '.JSHeapUsedSize'
```

## See also

- [trace](trace.md) - Capture a Chrome performance trace
- [metrics](metrics.md) - Get page performance metrics
