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

Returns an object containing an array of event listener objects.

| Field | Type | Description |
|-------|------|-------------|
| listeners | array | Array of event listener objects |
| listeners[].type | string | The event type (e.g., "click", "input") |
| listeners[].useCapture | boolean | Whether the listener uses capture phase |
| listeners[].passive | boolean | Whether the listener is passive |
| listeners[].once | boolean | Whether the listener fires only once |
| listeners[].scriptId | string | Script identifier where the listener is defined |
| listeners[].lineNumber | number | Line number in the script |
| listeners[].columnNumber | number | Column number in the script |

```json
{
  "listeners": [
    {"type":"click","useCapture":false,"passive":false,"once":false,"scriptId":"32","lineNumber":10,"columnNumber":2},
    {"type":"input","useCapture":false,"passive":true,"once":false,"scriptId":"32","lineNumber":15,"columnNumber":2}
  ]
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all event listeners on a button:

```
hubcap listeners '#submit'
```

List listeners and filter to just click handlers:

```
hubcap listeners '#submit' | jq '[.listeners[] | select(.type == "click")]'
```

## See also

- [dispatch](dispatch.md) - Dispatch an event on an element
- [query](query.md) - Query a DOM element
- [styles](styles.md) - Get computed CSS styles
