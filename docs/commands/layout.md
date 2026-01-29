# hubcap layout -- Get element layout with child positions

## When to use

Get element layout with child element positions. Use `bounds` for just the element's bounding box. Increase `--depth` for deeper nesting.

## Usage

```
hubcap layout <selector> [flags]
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --depth | int | 1 | Depth of children to include |

## Output

| Field | Type | Description |
|-------|------|-------------|
| tagName | string | The element's tag name |
| bounds | object | The element's bounding box (x, y, width, height) |
| children | array | Child elements with their own layout info |

```json
{
  "tagName": "DIV",
  "bounds": {"x": 0, "y": 0, "width": 800, "height": 600},
  "children": [
    {
      "tagName": "HEADER",
      "bounds": {"x": 0, "y": 0, "width": 800, "height": 60}
    }
  ]
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the layout of a container:

```
hubcap layout '#app'
```

Get a deeper layout tree and extract child widths:

```
hubcap layout '#app' --depth 3 | jq '.children[] | {tag: .tagName, width: .bounds.width}'
```

## See also

- [bounds](bounds.md) - Get just the element's bounding box
- [styles](styles.md) - Get computed CSS styles
- [domsnapshot](domsnapshot.md) - Get a full DOM snapshot
