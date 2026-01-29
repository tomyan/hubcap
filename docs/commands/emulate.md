# hubcap emulate

Emulate a mobile device with viewport, user agent, and touch.

## When to use

Emulate a mobile device with viewport, user agent, and touch capabilities. Use `viewport` to only change dimensions without setting user agent or touch. Use `useragent` to only change the user agent string.

## Usage

```
hubcap emulate <device>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `device` | string | Yes | Device name to emulate (e.g., `iPhone-12`, `Pixel-5`, `iPad-Pro`) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `device` | string | The device name that was emulated |
| `width` | number | Viewport width in pixels |
| `height` | number | Viewport height in pixels |
| `deviceScaleFactor` | number | Device pixel ratio |
| `mobile` | boolean | Whether mobile mode is enabled |

```json
{"device":"iPhone 12","width":390,"height":844,"deviceScaleFactor":3,"mobile":true}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Unknown device name | 1 | `emulate: unknown device: <name>` |
| Chrome not connected | 2 | `chrome: not connected` |
| Operation timed out | 3 | `timeout` |

## Examples

Emulate an iPhone 12:

```
hubcap emulate "iPhone 12"
```

Emulate a Pixel 5:

```
hubcap emulate "Pixel 5"
```

Emulate a device, navigate to a page, and take a screenshot:

```
hubcap emulate "iPhone 12" && hubcap goto 'https://example.com' && hubcap screenshot mobile.png
```

## See also

- [viewport](viewport.md) - Set viewport dimensions only
- [useragent](useragent.md) - Set user agent string only
- [tap](tap.md) - Tap an element (touch event)
- [swipe](swipe.md) - Swipe gesture on the page
- [media](media.md) - Emulate CSS media features
