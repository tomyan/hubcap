# hubcap intercept - Intercept and modify network requests or responses

## When to use

Intercept and modify network requests or responses in flight, such as replacing text in response bodies. Use `block` to simply block URLs without modification. Use `--disable` to stop interception when finished.

## Usage

```
hubcap intercept [flags]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pattern` | string | `"*"` | URL pattern to match |
| `--response` | bool | `false` | Intercept responses instead of requests |
| `--replace` | string | `""` | Text replacement as `old:new` |
| `--disable` | bool | `false` | Disable interception |

## Output

When enabling interception:

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Whether interception is active |
| `pattern` | string | URL pattern being matched |
| `response` | bool | Whether responses are intercepted |
| `replacement` | string | Text replacement rule |

```json
{"enabled":true,"pattern":"*","response":false,"replacement":"old:new"}
```

When disabling interception:

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Always `false` |

```json
{"enabled":false}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Invalid pattern | 1 | `error: invalid pattern "<value>"` |
| Invalid replace format | 1 | `error: invalid replace format "<value>"` |

## Examples

Intercept all requests matching an API pattern:

```bash
hubcap intercept --pattern "*/api/*"
```

Replace text in response bodies:

```bash
hubcap intercept --response --pattern "*.js" --replace "oldFunction:newFunction"
```

Disable interception:

```bash
hubcap intercept --disable
```

Intercept responses, modify them, then verify the change:

```bash
hubcap intercept --response --pattern "*/config.json" --replace "false:true" && hubcap goto "https://example.com" && hubcap eval "fetch('/config.json').then(r => r.json())"
```

## See also

- [block](block.md) - Block network requests by URL pattern
- [network](network.md) - Stream network requests and responses
- [responsebody](responsebody.md) - Get the response body for a captured request
