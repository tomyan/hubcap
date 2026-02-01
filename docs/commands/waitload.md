# hubcap waitload

Wait for the page load event to fire.

## When to use

Use `waitload` to block until the page load event fires. Use `goto --wait` for navigation plus load in one step. Use `waitidle` to wait for network quiet instead.

## Usage

```
hubcap waitload [--timeout <duration>]
```

## Arguments

None.

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field  | Type | Description                          |
|--------|------|--------------------------------------|
| loaded | bool | Whether the page load event fired    |

```json
{"loaded":true}
```

## Errors

| Condition                    | Exit code | Stderr                              |
|------------------------------|-----------|-------------------------------------|
| Chrome not connected         | 2         | `error: connecting to Chrome: ...`  |
| Load event not fired in time | 3         | `error: timeout`                    |

## Examples

Wait for page load with default timeout:

```
hubcap waitload
```

Wait with a custom timeout:

```
hubcap waitload --timeout 60s
```

Reload and wait for the page to finish loading (chaining):

```
hubcap reload && hubcap waitload
```

## See also

- [goto](goto.md) - Navigate to a URL (supports --wait)
- [waitnav](waitnav.md) - Wait for any navigation event
- [waitidle](waitidle.md) - Wait for network idle
- [waiturl](waiturl.md) - Wait for the URL to match a pattern
