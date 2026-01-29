# hubcap highlight -- Highlight an element for debugging

## When to use

Visually highlight an element for debugging. Use `--hide` to remove the highlight.

## Usage

```
hubcap highlight <selector> [flags]
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to highlight |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --hide | bool | false | Hide existing highlight |

## Output

When highlighting:

| Field | Type | Description |
|-------|------|-------------|
| highlighted | boolean | Whether the highlight was applied |
| selector | string | The selector that was used |

```json
{"highlighted":true,"selector":".sidebar"}
```

When hiding:

| Field | Type | Description |
|-------|------|-------------|
| hidden | boolean | Whether the highlight was removed |

```json
{"hidden":true}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Highlight a sidebar element:

```
hubcap highlight '.sidebar'
```

Highlight an element, take a screenshot, then remove the highlight:

```
hubcap highlight '.sidebar' && hubcap screenshot --path debug.png && hubcap highlight '.sidebar' --hide
```

## See also

- [bounds](bounds.md) - Get element bounding box
- [visible](visible.md) - Check if an element is visible
- [styles](styles.md) - Get computed CSS styles
