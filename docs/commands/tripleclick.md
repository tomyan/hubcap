# hubcap tripleclick -- Triple-click an element

## When to use

Triple-click to select a paragraph or line of text. Use `selection` to read the selected text afterward.

## Usage

```
hubcap tripleclick <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to triple-click |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| clicked | boolean | Whether the triple-click succeeded |
| selector | string | The selector that was used |

```json
{"clicked":true,"selector":"p.intro"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Triple-click to select a paragraph:

```
hubcap tripleclick 'p.intro'
```

Triple-click a paragraph, then read the selected text:

```
hubcap tripleclick 'p.intro' && hubcap selection
```

## See also

- [click](click.md) - Single-click an element
- [dblclick](dblclick.md) - Double-click an element
- [selection](selection.md) - Get the current text selection
