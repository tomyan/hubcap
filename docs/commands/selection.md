# hubcap selection

Get the currently selected text in the document.

## When to use

Use `selection` to retrieve whatever text is currently highlighted or selected in the page. Use `tripleclick` on an element first to select its text, or `caret` to get the cursor position within an input field.

## Usage

```
hubcap selection
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| text | string | The currently selected text |

```json
{"text":"selected text content"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the current selection:

```
hubcap selection
```

Extract just the selected text:

```
hubcap selection | jq -r '.text'
```

Select a paragraph by triple-clicking, then read the selection:

```
hubcap tripleclick "p.intro" && hubcap selection | jq -r '.text'
```

Verify that a keyboard shortcut (Ctrl+A) selects all text:

```
hubcap press "Control+a" && hubcap selection | jq -e '.text | length > 0'
```

## See also

- [caret](caret.md) - get the cursor position in an input field
- [tripleclick](tripleclick.md) - triple-click to select text
- [text](text.md) - get the text content of an element
