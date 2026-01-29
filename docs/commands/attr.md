# hubcap attr

Get a specific attribute value from an element.

## When to use

Use `attr` to read a single HTML attribute from the first element matching a selector. Use `query` to get all attributes of an element at once. Use `computed` to read CSS properties instead of HTML attributes.

## Usage

```
hubcap attr <selector> <attribute>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector for the target element |
| `attribute` | string | Yes | Name of the HTML attribute to read |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `selector` | string | The selector that was queried |
| `attribute` | string | The attribute name that was read |
| `value` | string | The attribute's value, or empty string if not present |

```json
{"selector":"#link","attribute":"href","value":"https://example.com"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the href of a link:

```
hubcap attr 'a.logo' 'href'
```

Get the src of an image:

```
hubcap attr '#hero-img' 'src'
```

Check an element's ARIA state:

```
hubcap attr '#menu' 'aria-expanded'
```

Read a data attribute then use it in a subsequent command:

```
hubcap attr '#product' 'data-id' | jq -r '.value' | xargs -I{} hubcap goto "https://example.com/api/products/{}"
```

## See also

- [query](query.md) - Get all attributes of an element at once
- [text](text.md) - Get the inner text of an element
- [html](html.md) - Get the full outer HTML of an element
- [computed](computed.md) - Get a computed CSS property value
- [value](value.md) - Get the value of a form input
