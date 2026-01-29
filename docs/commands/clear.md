# hubcap clear -- Clear an input field

## When to use

Clear an input field. Use `fill` to clear and type new text in one step. Use `setvalue` to set the value directly without simulating key events.

## Usage

```
hubcap clear <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the input to clear |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| cleared | boolean | Whether the clear succeeded |
| selector | string | The selector that was used |

```json
{"cleared":true,"selector":"#search"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: JS exception: Uncaught` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Clear a search input:

```
hubcap clear '#search'
```

Clear an input, then type new text:

```
hubcap clear '#email' && hubcap focus '#email' && hubcap type 'new@example.com'
```

## See also

- [fill](fill.md) - Focus, clear, and type in one step
- [setvalue](setvalue.md) - Set the value directly
- [focus](focus.md) - Focus an element
