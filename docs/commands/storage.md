# hubcap storage

Get, set, or clear localStorage entries.

## When to use

Use `storage` to read, write, or clear localStorage entries. Use `session` for sessionStorage. Use `cookies` for cookie management.

## Usage

```
hubcap storage <key> [value] | --clear
```

## Arguments

| Argument | Type   | Required             | Description                          |
|----------|--------|----------------------|--------------------------------------|
| key      | string | Yes (unless --clear) | The localStorage key to read or write |
| value    | string | No                   | Value to set; omit to read           |

## Flags

| Flag    | Type | Default | Description              |
|---------|------|---------|--------------------------|
| --clear | bool | false   | Clear all localStorage   |

## Output

**Get mode** (key only):

| Field | Type   | Description        |
|-------|--------|--------------------|
| key   | string | The storage key    |
| value | string | The stored value   |

```json
{"key":"theme","value":"dark"}
```

**Set mode** (key and value):

| Field | Type   | Description          |
|-------|--------|----------------------|
| key   | string | The storage key      |
| value | string | The value that was set |
| set   | bool   | Whether the value was set |

```json
{"key":"theme","value":"light","set":true}
```

**Clear mode** (`--clear`):

| Field   | Type | Description                      |
|---------|------|----------------------------------|
| cleared | bool | Whether localStorage was cleared |

```json
{"cleared":true}
```

## Errors

| Condition            | Exit code | Stderr                        |
|----------------------|-----------|-------------------------------|
| Missing key argument | 1         | `usage: hubcap storage <key> [value] \| --clear`|
| Chrome not connected | 2         | `error: connecting to Chrome: ...` |
| Operation timeout    | 3         | `error: timeout`              |

## Examples

Read a value:

```
hubcap storage theme
```

Set a value:

```
hubcap storage theme light
```

Clear all localStorage:

```
hubcap storage --clear
```

Set a value and verify it was stored (chaining):

```
hubcap storage theme dark && hubcap storage theme
```

## See also

- [session](session.md) - Get, set, or clear sessionStorage entries
- [cookies](cookies.md) - Manage browser cookies
- [clipboard](clipboard.md) - Read or write the system clipboard
