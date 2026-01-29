# hubcap swipe

Perform a touch swipe gesture on an element.

## When to use

Use `swipe` to simulate a directional touch swipe for mobile emulation testing. Use `scroll` for desktop-style pixel scrolling instead. Pair with `emulate` to set a mobile device profile for full mobile simulation.

## Usage

```
hubcap swipe <selector> <direction>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the element to swipe on |
| `direction` | string | Yes | Swipe direction: `left`, `right`, `up`, or `down` |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `swiped` | boolean | Whether the swipe succeeded |
| `direction` | string | The swipe direction that was performed |
| `selector` | string | The selector that was swiped |

```json
{"swiped":true,"direction":"left","selector":".carousel"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Swipe left on a carousel:

```
hubcap swipe '.carousel' left
```

Swipe down to refresh:

```
hubcap swipe '#feed' down
```

Dismiss a notification by swiping right:

```
hubcap swipe '.notification' right
```

Emulate a mobile device then swipe through a gallery:

```
hubcap emulate 'iPhone 14' && hubcap swipe '.gallery' left && hubcap swipe '.gallery' left
```

## See also

- [pinch](pinch.md) - Pinch zoom gesture
- [tap](tap.md) - Touch tap gesture
- [scroll](scroll.md) - Desktop pixel scrolling
- [emulate](emulate.md) - Emulate a mobile device
