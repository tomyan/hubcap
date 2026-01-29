# hubcap offline

Enable or disable the browser's offline mode.

## When to use

Simulate a complete network disconnection to test how a page behaves when offline, such as verifying service worker caching or offline fallback UI. Use `throttle` for slow-but-connected simulation rather than a full disconnect.

## Usage

```
hubcap offline <true|false>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `value` | boolean | Yes | `true` to enable offline mode, `false` to disable it |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `offline` | boolean | Whether offline mode is now active |

```json
{"offline":true}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Missing argument | 1 | `usage: hubcap offline <true\|false>` |
| Invalid argument | 1 | `error: invalid value, use 'true' or 'false'` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Go offline:

```
hubcap offline true
```

Restore connectivity:

```
hubcap offline false
```

Go offline, reload the page, and check that the service worker serves cached content:

```
hubcap offline true && hubcap reload && hubcap waitload && hubcap text "#status"
```

## See also

- [throttle](throttle.md) - Simulate slow network conditions
- [block](block.md) - Block specific URLs by pattern
- [network](network.md) - Monitor network traffic
