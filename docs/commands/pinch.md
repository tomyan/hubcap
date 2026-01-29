# hubcap pinch

Perform a pinch zoom gesture on an element.

## When to use

Use `pinch` to simulate a two-finger pinch gesture for mobile emulation testing. Use `swipe` for directional swipe gestures. Pair with `emulate` to set a mobile device profile for full mobile simulation.

## Usage

```
hubcap pinch <selector> <direction>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the element to pinch |
| `direction` | string | Yes | Pinch direction: `in` (zoom out) or `out` (zoom in) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `pinched` | boolean | Whether the pinch succeeded |
| `direction` | string | The pinch direction that was performed |
| `selector` | string | The selector that was pinched |

```json
{"pinched":true,"direction":"out","selector":"#map"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: element not found: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Pinch out to zoom into a map:

```
hubcap pinch '#map' out
```

Pinch in to zoom out of an image:

```
hubcap pinch '.photo-viewer' in
```

Emulate a mobile device then pinch to zoom:

```
hubcap emulate 'iPhone 14' && hubcap pinch '#map' out
```

## See also

- [swipe](swipe.md) - Touch swipe gesture
- [tap](tap.md) - Touch tap gesture
- [emulate](emulate.md) - Emulate a mobile device
