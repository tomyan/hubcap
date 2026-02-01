# hubcap uncheck -- Uncheck a checkbox

## When to use

Uncheck a checkbox. Idempotent -- does nothing if already unchecked. Use `check` to check.

## Usage

```
hubcap uncheck <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the checkbox to uncheck |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| unchecked | boolean | Whether the checkbox is now unchecked |
| selector | string | The selector that was used |

```json
{"unchecked":true,"selector":"#newsletter"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: JS exception: <message>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Uncheck a newsletter opt-in checkbox:

```
hubcap uncheck '#newsletter'
```

Uncheck a checkbox and verify it is unchecked:

```
hubcap uncheck '#newsletter' && hubcap eval 'document.querySelector("#newsletter").checked'
```

## See also

- [check](check.md) - Check a checkbox
- [click](click.md) - Single-click an element
