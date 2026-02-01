# hubcap text -- Get the inner text of an element

## When to use

Get the inner text of an element. Use `html` for the full outer HTML. Use `value` for form input values.

## Usage

```
hubcap text <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to read |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| selector | string | The selector that was used |
| text | string | The inner text of the element |

```json
{"selector":"h1","text":"Welcome to Hubcap"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the text of a heading:

```
hubcap text 'h1'
```

Get button text and pipe it to another command:

```
hubcap text '.status-message' | jq -r '.text'
```

## See also

- [html](html.md) - Get outer HTML of an element
- [value](value.md) - Get the value of an input element
- [attr](attr.md) - Get an attribute of an element
- [query](query.md) - Query a DOM element
