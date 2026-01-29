# hubcap focus -- Focus an element

## When to use

Focus an element before typing. Required before `type` for non-input elements. Use `fill` to focus, clear, and type in one step.

## Usage

```
hubcap focus <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to focus |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| focused | boolean | Whether the focus succeeded |
| selector | string | The selector that was used |

```json
{"focused":true,"selector":"#username"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Focus an input field:

```
hubcap focus '#username'
```

Focus a contenteditable div, then type into it:

```
hubcap focus '[contenteditable]' && hubcap type 'Hello, world!'
```

## See also

- [fill](fill.md) - Focus, clear, and type in one step
- [type](type.md) - Type text into the focused element
- [click](click.md) - Single-click an element
- [clear](clear.md) - Clear an input field
