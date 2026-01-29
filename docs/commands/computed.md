# hubcap computed

Get a single computed CSS property value for an element.

## When to use

Use `computed` to read one resolved CSS property value from the first element matching a selector. Use `styles` to get all computed styles at once instead of querying properties individually.

## Usage

```
hubcap computed <selector> <property>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector for the target element |
| `property` | string | Yes | CSS property name to read (e.g. `color`, `display`, `font-size`) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `selector` | string | The selector that was queried |
| `property` | string | The CSS property that was queried |
| `value` | string | Computed value of the property |

```json
{"selector":"h1","property":"color","value":"rgb(255, 0, 0)"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the font size of a heading:

```
hubcap computed 'h1' 'font-size'
```

Check the display property:

```
hubcap computed '#sidebar' 'display'
```

Read the background color:

```
hubcap computed '.alert' 'background-color'
```

Verify an element is hidden by checking its display value:

```
hubcap click '#toggle' && hubcap computed '#panel' 'display'
```

## See also

- [styles](styles.md) - Get all computed CSS styles at once
- [attr](attr.md) - Get an HTML attribute value
- [visible](visible.md) - Check whether an element is visible
- [bounds](bounds.md) - Get position and dimensions of an element
