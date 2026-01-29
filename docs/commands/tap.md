# hubcap tap -- Touch-tap an element

## When to use

Touch-tap an element for mobile emulation. Use `click` for desktop mouse clicks. Pair with `emulate` for full mobile simulation.

## Usage

```
hubcap tap <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to tap |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| tapped | boolean | Whether the tap succeeded |
| selector | string | The selector that was used |

```json
{"tapped":true,"selector":"#submit"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Tap a button in a mobile-emulated page:

```
hubcap tap '#submit'
```

Emulate a mobile device, then tap a hamburger menu:

```
hubcap emulate 'iPhone 15' && hubcap tap '.hamburger-menu'
```

## See also

- [click](click.md) - Single-click an element (desktop)
- [swipe](swipe.md) - Swipe gesture on an element
- [pinch](pinch.md) - Pinch gesture on an element
- [emulate](emulate.md) - Emulate a mobile device
