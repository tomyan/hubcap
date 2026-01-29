# hubcap listeners -- List event listeners on an element

## When to use

List event listeners attached to an element. Useful for debugging event handling and understanding which events an element responds to.

## Usage

```
hubcap listeners <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to inspect |

## Flags

None.

## Output

Returns an array of event listener objects.

| Field | Type | Description |
|-------|------|-------------|
| type | string | The event type (e.g., "click", "input") |
| useCapture | boolean | Whether the listener uses capture phase |
| passive | boolean | Whether the listener is passive |
| once | boolean | Whether the listener fires only once |
| handler | string | The handler function source |

```json
[
  {"type":"click","useCapture":false,"passive":false,"once":false,"handler":"function() { ... }"},
  {"type":"input","useCapture":false,"passive":true,"once":false,"handler":"function(e) { ... }"}
]
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all event listeners on a button:

```
hubcap listeners '#submit'
```

List listeners and filter to just click handlers:

```
hubcap listeners '#submit' | jq '[.[] | select(.type == "click")]'
```

## See also

- [dispatch](dispatch.md) - Dispatch an event on an element
- [query](query.md) - Query a DOM element
- [styles](styles.md) - Get computed CSS styles
