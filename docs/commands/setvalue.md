# hubcap setvalue

Set an input element's value directly without simulating keystrokes.

## When to use

Use `setvalue` to programmatically assign a value to an input element, bypassing individual keystroke events. This is useful for sliders, date pickers, hidden inputs, and other elements where simulated typing is unnecessary or impractical. Use `fill` for realistic user input simulation that fires keystroke events.

## Usage

```
hubcap setvalue <selector> <value>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the input element |
| `value` | string | Yes | Value to set on the element |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `set` | boolean | Whether the value was set |
| `selector` | string | The selector of the element |
| `value` | string | The value that was set |

```json
{"set":true,"selector":"#slider","value":"75"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Set a range slider value:

```
hubcap setvalue '#slider' '75'
```

Set a hidden input value:

```
hubcap setvalue '[name="token"]' 'abc123'
```

Set a date input:

```
hubcap setvalue '#date' '2025-01-15'
```

Set a value then dispatch a change event to trigger listeners:

```
hubcap setvalue '#config' 'dark' && hubcap dispatch '#config' change
```

## See also

- [fill](fill.md) - Fill an input by typing with keystroke events
- [value](value.md) - Get the current value of an input
- [clear](clear.md) - Clear an input value
