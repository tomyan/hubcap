# hubcap query -- Query a DOM element

## When to use

Query a DOM element to get its tag name and attributes. Use `text` or `html` to get content. Use `exists` for a boolean check.

## Usage

```
hubcap query <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to query |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| nodeId | number | The DOM node ID |
| tagName | string | The element's tag name in uppercase |
| attributes | object | Key-value pairs of the element's attributes |

```json
{"nodeId":123,"tagName":"DIV","attributes":{"class":"container","id":"main"}}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 0 | None (returns `{"nodeId":0}`) |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Query an element to inspect its attributes:

```
hubcap query '#main'
```

Query an element and extract its class attribute:

```
hubcap query '.hero' | jq -r '.attributes.class'
```

## See also

- [text](text.md) - Get inner text of an element
- [html](html.md) - Get outer HTML of an element
- [attr](attr.md) - Get a specific attribute of an element
- [exists](exists.md) - Check if an element exists
- [count](count.md) - Count matching elements
