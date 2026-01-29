# hubcap fill

Clear an input and type new text in one step.

## When to use

Use `fill` to replace the contents of an input field with new text, simulating realistic user input with keystrokes. Use `type` to append text without clearing first. Use `setvalue` to set the value directly without firing individual keystroke events.

## Usage

```
hubcap fill <selector> <text>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the input element |
| `text` | string | Yes | Text to fill into the input |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `filled` | boolean | Whether the fill succeeded |
| `selector` | string | The selector that was filled |
| `text` | string | The text that was entered |

```json
{"filled":true,"selector":"#email","text":"user@example.com"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Fill an email input:

```
hubcap fill '#email' 'user@example.com'
```

Fill a search box:

```
hubcap fill '[name="q"]' 'hubcap CLI'
```

Fill a login form and submit it:

```
hubcap fill '#email' 'user@example.com' && hubcap fill '#password' 'secret' && hubcap click '#submit'
```

## See also

- [clear](clear.md) - Clear an input field
- [type](type.md) - Type text keystroke by keystroke without clearing
- [setvalue](setvalue.md) - Set a value directly without typing
- [focus](focus.md) - Focus an element
- [value](value.md) - Get the current value of an input
