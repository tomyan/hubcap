# hubcap value -- Get the value of an input element

## When to use

Get the current value of an input, textarea, or select element. Use `text` for non-form elements. Use `setvalue` to change the value.

## Usage

```
hubcap value <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the input, textarea, or select element |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| selector | string | The selector that was used |
| value | string | The current value of the element |

```json
{"selector":"#email","value":"user@example.com"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: selector "<sel>": element not found` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the value of an email input:

```
hubcap value '#email'
```

Fill an input, then verify its value:

```
hubcap fill '#email' 'test@example.com' && hubcap value '#email' | jq -r '.value'
```

## See also

- [text](text.md) - Get inner text of an element
- [setvalue](setvalue.md) - Set the value directly
- [fill](fill.md) - Focus, clear, and type in one step
- [attr](attr.md) - Get an attribute of an element
