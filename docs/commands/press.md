# hubcap press

Press a key or key combination.

## When to use

Press a key or key combination. Supports modifiers: `Ctrl+`, `Alt+`, `Shift+`, `Meta+`. Use `type` for text input instead. Named keys: Enter, Tab, Escape, Backspace, Delete, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Home, End, PageUp, PageDown, Space.

## Usage

```
hubcap press <key>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `key` | string | Yes | Key name or combination (e.g., `Enter`, `Ctrl+a`, `Ctrl+Shift+n`) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `pressed` | boolean | Whether the key press succeeded |
| `key` | string | The key or combination that was pressed |

```json
{"pressed":true,"key":"Enter"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `chrome: not connected` |
| Operation timed out | 3 | `timeout` |

## Examples

Press Enter to submit a form:

```
hubcap press Enter
```

Select all text:

```
hubcap press Ctrl+a
```

Press Escape to close a dialog:

```
hubcap press Escape
```

Focus an input, select all existing text, then type a replacement:

```
hubcap focus '#email' && hubcap press Ctrl+a && hubcap type 'new@example.com'
```

## See also

- [type](type.md) - Type text keystroke by keystroke
- [fill](fill.md) - Clear and type into an input field
- [focus](focus.md) - Focus an element before pressing keys
