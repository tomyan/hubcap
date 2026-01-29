# hubcap caret -- Get the caret position in an input

## When to use

Get the cursor/caret position within an input or textarea. Use `selection` for document-level text selection.

## Usage

```
hubcap caret <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the input or textarea |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| position | number | Current caret position (character index) |
| selectionStart | number | Start index of the selection range |
| selectionEnd | number | End index of the selection range |

```json
{"position":5,"selectionStart":5,"selectionEnd":5}
```

When text is selected within the input:

```json
{"position":10,"selectionStart":3,"selectionEnd":10}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the caret position in a text input:

```
hubcap caret '#search'
```

Focus an input, type some text, then check the caret position:

```
hubcap fill '#editor' 'Hello world' && hubcap caret '#editor' | jq -r '.position'
```

## See also

- [selection](selection.md) - Get document-level text selection
- [focus](focus.md) - Focus an element
- [fill](fill.md) - Focus, clear, and type in one step
- [value](value.md) - Get the value of an input
