# hubcap find

Find text occurrences on the page.

## When to use

Find text occurrences on the page. Returns match count and positions. Use `waittext` to wait for text to appear instead of checking once.

## Usage

```
hubcap find <text>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `text` | string | Yes | Text string to search for on the page |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `found` | boolean | `true` if at least one occurrence exists |
| `count` | number | Total number of occurrences on the page |
| `text` | string | The text string that was searched for |

```json
{"found":true,"count":3,"text":"hello"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Search for a word on the page:

```
hubcap find "Login"
```

Check if an error message appears:

```
hubcap find "Something went wrong"
```

Check for text and conditionally take action based on the result:

```
hubcap find "Success" | jq -e '.found' && hubcap screenshot success.png
```

## See also

- [waittext](waittext.md) - Wait for text to appear on the page
- [text](text.md) - Get the inner text of a specific element
- [exists](exists.md) - Check if an element exists by CSS selector
