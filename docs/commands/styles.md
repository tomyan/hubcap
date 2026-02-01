# hubcap styles -- Get all computed CSS styles of an element

## When to use

Get all computed CSS styles of an element. Use `computed` for a single CSS property value.

## Usage

```
hubcap styles <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| selector | string | The selector that was used |
| styles | object | All computed CSS property-value pairs |

```json
{"selector":".btn","styles":{"color":"rgb(0, 0, 0)","display":"block","font-size":"16px"}}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get all styles of a button:

```
hubcap styles '.btn'
```

Get styles and extract just the background color:

```
hubcap styles '.btn' | jq -r '.styles["background-color"]'
```

## See also

- [computed](computed.md) - Get a single computed CSS property
- [visible](visible.md) - Check if an element is visible
- [bounds](bounds.md) - Get element bounding box
- [highlight](highlight.md) - Highlight an element for debugging
