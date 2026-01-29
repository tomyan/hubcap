# hubcap block

Block network requests matching URL patterns.

## When to use

Block network requests that match one or more URL patterns, such as ads, tracking scripts, or specific resource types. Use `--disable` to remove all active block rules. Use `intercept` instead if you need to modify request or response content rather than simply blocking.

## Usage

```
hubcap block <pattern>... [--disable]
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `pattern` | string | Yes | One or more URL patterns to block. Supports `*` wildcards. |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--disable` | bool | `false` | Disable URL blocking and remove all block rules |

## Output

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Whether blocking is active |
| `patterns` | string[] | List of active block patterns (when enabled) |

When rules are set:

```json
{"enabled":true,"patterns":["*.css","*.png"]}
```

When disabled:

```json
{"enabled":false}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| No patterns provided and `--disable` not set | 1 | `usage: hubcap block <pattern>... [--disable]` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Block ad and tracking requests:

```
hubcap block "*ads*" "*tracking*" "*analytics*"
```

Block all CSS and image files:

```
hubcap block "*.css" "*.png" "*.jpg"
```

Remove all block rules:

```
hubcap block --disable
```

Block third-party scripts, then take a screenshot to measure visual impact:

```
hubcap block "*cdn.third-party.com*" && hubcap screenshot --output page-no-thirdparty.png
```

## See also

- [intercept](intercept.md) - Intercept and modify requests or responses
- [throttle](throttle.md) - Simulate slow network conditions
- [network](network.md) - Monitor network traffic
