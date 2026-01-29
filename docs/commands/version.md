# hubcap version

Print the browser version, protocol version, user-agent string, and V8 engine version.

## When to use

Use `version` to check what browser and protocol you are connected to, or to verify capabilities before running commands that depend on a specific Chrome version. Use `info` instead if you need page-level details like title or URL.

## Usage

```
hubcap version
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| browser | string | Browser name and version (e.g. `Chrome/120.0.6099.109`) |
| protocol | string | DevTools protocol version (e.g. `1.3`) |
| userAgent | string | Full user-agent string reported by the browser |
| v8 | string | V8 JavaScript engine version |

```json
{
  "browser": "Chrome/120.0.6099.109",
  "protocol": "1.3",
  "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.109 Safari/537.36",
  "v8": "12.0.267.8"
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Print the browser version:

```
hubcap version
```

Check whether the connected browser is at least Chrome 120:

```
hubcap version | jq -r '.browser' | grep -q '12[0-9]\.' && echo "OK"
```

Use the protocol version in a script that selects behaviour by version:

```
PROTO=$(hubcap version | jq -r '.protocol')
echo "Protocol: $PROTO"
```

## See also

- [info](info.md) - get combined page information (title, URL, meta)
- [tabs](tabs.md) - list open browser tabs
