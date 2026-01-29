# hubcap dispatch

Dispatch a custom DOM event on an element.

## When to use

Use `dispatch` to fire a DOM event on an element when built-in commands like `click`, `fill`, or `setvalue` do not trigger the event handlers your page requires. This is useful for custom events or for manually triggering `change`, `input`, or `submit` events.

## Usage

```
hubcap dispatch <selector> <eventType>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the target element |
| `eventType` | string | Yes | Name of the event to dispatch (e.g. `change`, `input`, `submit`) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `dispatched` | boolean | Whether the event was dispatched |
| `eventType` | string | The event type that was dispatched |
| `selector` | string | The selector of the target element |

```json
{"dispatched":true,"eventType":"change","selector":"#config"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Dispatch a change event:

```
hubcap dispatch '#config' change
```

Dispatch a custom application event:

```
hubcap dispatch '.widget' app-refresh
```

Trigger an input event:

```
hubcap dispatch '#search' input
```

Set a value then dispatch change to notify listeners:

```
hubcap setvalue '#theme' 'dark' && hubcap dispatch '#theme' change
```

## See also

- [click](click.md) - Click an element (dispatches click event)
- [fill](fill.md) - Fill an input (dispatches keystroke events)
- [listeners](listeners.md) - List event listeners on an element
