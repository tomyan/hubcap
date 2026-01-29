# hubcap metrics

Get performance metrics from the browser.

## When to use

Use `metrics` to get a quick snapshot of performance data like JavaScript heap size, DOM node count, and layout count. This is a lightweight health check without the overhead of full profiling. Use `trace` for detailed performance traces or `heapsnapshot` for memory analysis.

## Usage

```
hubcap metrics
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| metrics | object | Key-value map of performance metric names to numeric values |
| metrics.JSHeapUsedSize | number | Bytes of JavaScript heap currently in use |
| metrics.JSHeapTotalSize | number | Total bytes allocated for the JavaScript heap |
| metrics.Nodes | number | Number of DOM nodes in the document |
| metrics.LayoutCount | number | Number of layout operations performed |
| metrics.Timestamp | number | Timestamp of the measurement |

```json
{
  "metrics": {
    "Timestamp": 1234567.89,
    "JSHeapUsedSize": 5242880,
    "JSHeapTotalSize": 8388608,
    "Nodes": 150,
    "LayoutCount": 12,
    "RecalcStyleCount": 8,
    "Documents": 1,
    "Frames": 1
  }
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get performance metrics:

```
hubcap metrics
```

Check JS heap usage in megabytes:

```
hubcap metrics | jq '.metrics.JSHeapUsedSize / 1048576 | round | tostring + " MB"'
```

Monitor DOM node count before and after an action:

```
BEFORE=$(hubcap metrics | jq '.metrics.Nodes')
hubcap click "#load-more"
hubcap waitidle
AFTER=$(hubcap metrics | jq '.metrics.Nodes')
echo "Nodes added: $(( AFTER - BEFORE ))"
```

## See also

- [heapsnapshot](heapsnapshot.md) - capture a full heap snapshot
- [trace](trace.md) - record a performance trace
- [coverage](coverage.md) - get JavaScript code coverage
