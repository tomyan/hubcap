# hubcap check -- Check a checkbox

## When to use

Check a checkbox. Idempotent -- does nothing if already checked. Use `uncheck` to uncheck.

## Usage

```
hubcap check <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the checkbox to check |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| checked | boolean | Whether the checkbox is now checked |
| selector | string | The selector that was used |

```json
{"checked":true,"selector":"#agree-tos"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Check the terms-of-service checkbox:

```
hubcap check '#agree-tos'
```

Check a checkbox, then submit the form:

```
hubcap check '#agree-tos' && hubcap click '#submit'
```

## See also

- [uncheck](uncheck.md) - Uncheck a checkbox
- [click](click.md) - Single-click an element
- [visible](visible.md) - Check if an element is visible
