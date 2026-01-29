# hubcap select

Select a dropdown option by its value attribute.

## When to use

Use `select` to choose an option from a native `<select>` dropdown by matching the `<option>` value attribute. Use `click` on custom dropdown elements that are not native `<select>` elements.

## Usage

```
hubcap select <selector> <value>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the `<select>` element |
| `value` | string | Yes | Value attribute of the `<option>` to select |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `selected` | boolean | Whether the selection succeeded |
| `selector` | string | The selector of the dropdown |
| `value` | string | The value that was selected |

```json
{"selected":true,"selector":"#country","value":"us"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Select a country from a dropdown:

```
hubcap select '#country' 'us'
```

Select a size option:

```
hubcap select '[name="size"]' 'large'
```

Select an option then verify the value was set:

```
hubcap select '#country' 'us' && hubcap value '#country'
```

## See also

- [fill](fill.md) - Fill a text input
- [value](value.md) - Get the current value of an input
- [check](check.md) - Check a checkbox
- [forms](forms.md) - Get all form elements on the page
