# hubcap viewport

Set the browser viewport size.

## When to use

Set the browser viewport to specific width and height in pixels. Use `emulate` for full device emulation including user agent and device pixel ratio.

## Usage

```
hubcap viewport <width> <height>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `width` | int | Yes | Viewport width in pixels |
| `height` | int | Yes | Viewport height in pixels |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `width` | number | The viewport width that was set |
| `height` | number | The viewport height that was set |

```json
{"width":1920,"height":1080}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Invalid width (non-numeric) | 1 | `invalid width: <value>` |
| Invalid height (non-numeric) | 1 | `invalid height: <value>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Set a 1080p viewport:

```
hubcap viewport 1920 1080
```

Set a mobile-sized viewport:

```
hubcap viewport 375 667
```

Set a viewport size and then take a screenshot at that resolution:

```
hubcap viewport 1440 900 && hubcap screenshot desktop.png
```

## See also

- [emulate](emulate.md) - Emulate a full device profile with viewport, user agent, and touch
- [screenshot](screenshot.md) - Take a screenshot of the page
- [pdf](pdf.md) - Generate a PDF of the page
