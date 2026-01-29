# hubcap dblclick -- Double-click an element

## When to use

Double-click an element. Use `click` for a single click, or `tripleclick` to select an entire paragraph or line of text.

## Usage

```
hubcap dblclick <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to double-click |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| clicked | boolean | Whether the double-click succeeded |
| selector | string | The selector that was used |

```json
{"clicked":true,"selector":"#editable"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Double-click a list item to edit it:

```
hubcap dblclick '.item-label'
```

Double-click, then read the resulting text input value:

```
hubcap dblclick '.editable-cell' && hubcap value '.editable-cell input'
```

## See also

- [click](click.md) - Single-click an element
- [rightclick](rightclick.md) - Right-click an element
- [tripleclick](tripleclick.md) - Triple-click an element
- [hover](hover.md) - Hover over an element
