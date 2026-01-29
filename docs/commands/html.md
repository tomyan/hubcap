# hubcap html -- Get the outer HTML of an element

## When to use

Get the outer HTML of an element. Use `text` for just the text content. Use `source` for the full page HTML.

## Usage

```
hubcap html <selector>
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
| html | string | The outer HTML of the element |

```json
{"selector":"#main","html":"<div id=\"main\"><p>Hello</p></div>"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the HTML of a container:

```
hubcap html '#main'
```

Get a nav element's HTML and extract it with jq:

```
hubcap html 'nav' | jq -r '.html'
```

## See also

- [text](text.md) - Get inner text of an element
- [source](source.md) - Get full page HTML source
- [query](query.md) - Query a DOM element
- [attr](attr.md) - Get an attribute of an element
