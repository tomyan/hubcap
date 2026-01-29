# hubcap domsnapshot

Capture a complete DOM snapshot including computed styles and layout information.

## When to use

Use `domsnapshot` to get a comprehensive snapshot of the DOM that includes computed styles and layout data. This is heavier than `source` but provides full layout information. Use `source` if you only need the raw HTML.

## Usage

```
hubcap domsnapshot
```

## Arguments

None.

## Flags

None.

## Output

Returns a full DOM snapshot object containing document nodes, layout information, and computed styles.

| Field | Type | Description |
|-------|------|-------------|
| documents | array | Array of document snapshot objects |
| documents[].nodes | object | DOM node tree data |
| documents[].layout | object | Layout tree data with bounds and styles |
| strings | array | String table referenced by index in the snapshot |

```json
{
  "documents": [
    {
      "nodes": {
        "nodeName": [0, 1, 2],
        "nodeValue": [-1, -1, 3]
      },
      "layout": {
        "nodeIndex": [0, 1],
        "bounds": [[0, 0, 800, 600], [8, 8, 784, 20]]
      }
    }
  ],
  "strings": ["html", "head", "body", "Hello"]
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Capture a DOM snapshot:

```
hubcap domsnapshot
```

Save the snapshot to a file for offline analysis:

```
hubcap domsnapshot > snapshot.json
```

Compare DOM snapshots before and after an interaction:

```
hubcap domsnapshot > before.json && hubcap click "#expand-button" && hubcap domsnapshot > after.json && diff <(jq -S . before.json) <(jq -S . after.json)
```

## See also

- [source](source.md) - get the raw HTML source (lighter weight)
- [a11y](a11y.md) - get the accessibility tree
- [layout](layout.md) - get layout information for specific elements
