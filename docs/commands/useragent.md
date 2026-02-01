# hubcap useragent

Override the browser's user agent string.

## When to use

Override the browser user agent string to test server-side user agent detection or mimic a specific browser or crawler. Use `emulate` for full device emulation including viewport and touch support alongside the user agent.

## Usage

```
hubcap useragent <string>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `string` | string | Yes | The user agent string to set |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `userAgent` | string | The user agent string that was set |

```json
{"userAgent":"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Missing argument | 1 | `usage: hubcap useragent <string>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Pretend to be Googlebot:

```
hubcap useragent "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
```

Pretend to be a specific desktop browser:

```
hubcap useragent "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
```

Set the user agent, navigate, and verify the server responded differently:

```
hubcap useragent "Googlebot/2.1" && hubcap goto https://example.com && hubcap text "body"
```

## See also

- [emulate](emulate.md) - Emulate a full device profile including viewport and touch
- [viewport](viewport.md) - Set viewport dimensions
- [version](version.md) - Print browser version information
