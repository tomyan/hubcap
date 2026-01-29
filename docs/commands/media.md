# hubcap media

Emulate CSS media features such as color scheme, reduced motion, and forced colors.

## When to use

Emulate CSS media features like dark mode, reduced motion, or forced colors to test responsive design without changing OS settings. At least one flag is required per invocation. Use `emulate` for full device emulation including viewport, user agent, and touch support.

## Usage

```
hubcap media [flags]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--color-scheme` | string | `""` | Set `prefers-color-scheme`: `light` or `dark` |
| `--reduced-motion` | string | `""` | Set `prefers-reduced-motion`: `reduce` or `no-preference` |
| `--forced-colors` | string | `""` | Set `forced-colors`: `active` or `none` |

## Output

| Field | Type | Description |
|-------|------|-------------|
| `colorScheme` | string | The applied color scheme value |
| `reducedMotion` | string | The applied reduced motion value |
| `forcedColors` | string | The applied forced colors value |

Only fields for the flags you specify appear in the output.

```json
{"colorScheme":"dark","reducedMotion":"","forcedColors":""}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| No flag specified | 1 | `error: at least one media flag required` |
| Chrome not connected | 2 | `error: chrome connection failed` |
| Timeout waiting for response | 3 | `error: timeout` |

## Examples

Enable dark mode:

```
hubcap media --color-scheme dark
```

Enable reduced motion:

```
hubcap media --reduced-motion reduce
```

Combine multiple features:

```
hubcap media --color-scheme dark --reduced-motion reduce --forced-colors active
```

Switch to dark mode and take a screenshot to compare themes:

```
hubcap media --color-scheme dark && hubcap screenshot dark-mode.png
```

## See also

- [emulate](emulate.md) - Emulate a full device profile including viewport and touch
- [viewport](viewport.md) - Set viewport dimensions
- [screenshot](screenshot.md) - Capture a screenshot of the page
