# hubcap throttle

Simulate slow network conditions using a throttling preset.

## When to use

Simulate slow network conditions to test how a page performs under constrained bandwidth and high latency. Use `offline` to fully disconnect the browser from the network. Use `block` to block specific URLs rather than slowing all traffic.

## Usage

```
hubcap throttle <preset> | --disable
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `preset` | string | Yes (when not disabling) | Throttling profile to apply (e.g. `3g`, `slow3g`, `fast3g`, `4g`, `wifi`) |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--disable` | bool | `false` | Disable network throttling and restore normal speed |

## Output

| Field | Type | Description |
|-------|------|-------------|
| `preset` | string | Name of the applied preset (when enabling) |
| `enabled` | boolean | Whether throttling is active (when enabling) |
| `disabled` | boolean | Whether throttling was disabled (when disabling) |

When a preset is applied:

```json
{"preset":"3g","enabled":true}
```

When disabled:

```json
{"disabled":true}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Unknown preset name | 1 | `error: unknown preset "name"` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Simulate a 3G connection:

```
hubcap throttle 3g
```

Simulate a slow 3G connection:

```
hubcap throttle slow3g
```

Disable throttling:

```
hubcap throttle --disable
```

Throttle the network, load a page, and capture performance metrics:

```
hubcap throttle slow3g && hubcap goto https://example.com && hubcap metrics
```

## See also

- [offline](offline.md) - Toggle offline mode for full network disconnection
- [block](block.md) - Block specific URLs by pattern
- [network](network.md) - Monitor network traffic
