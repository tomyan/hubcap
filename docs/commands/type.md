# hubcap type

Type text keystroke by keystroke into the currently focused element.

## When to use

Type text keystroke by keystroke into the currently focused element. Does NOT clear existing content first. Use `fill` to clear and then type. Use `press` for special keys like Enter or Tab.

## Usage

```
hubcap type <text>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `text` | string | Yes | Text to type, with optional escape sequences (`\n` for Enter, `\t` for Tab, `\\` for literal backslash) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `typed` | boolean | Whether the typing succeeded |
| `text` | string | The text that was typed |

```json
{"typed":true,"text":"hello world"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Type a simple string:

```
hubcap type 'hello world'
```

Type text and press Enter:

```
hubcap type 'search query\n'
```

Type text with a Tab between fields:

```
hubcap type 'first\tlast'
```

Focus an input first, then type into it:

```
hubcap focus '#search' && hubcap type 'hubcap cli'
```

## See also

- [fill](fill.md) - Clear and type into an input field
- [press](press.md) - Press a single key or key combination
- [focus](focus.md) - Focus an element before typing
- [clear](clear.md) - Clear an input field
