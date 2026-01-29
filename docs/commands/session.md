# hubcap session

Get, set, or clear sessionStorage entries.

## When to use

Use `session` to read, write, or clear sessionStorage entries. Use `storage` for localStorage. Session storage is cleared when the tab is closed.

## Usage

```
hubcap session <key> [value] | --clear
```

## Arguments

| Argument | Type   | Required             | Description                            |
|----------|--------|----------------------|----------------------------------------|
| key      | string | Yes (unless --clear) | The sessionStorage key to read or write |
| value    | string | No                   | Value to set; omit to read             |

## Flags

| Flag    | Type | Default | Description                |
|---------|------|---------|----------------------------|
| --clear | bool | false   | Clear all sessionStorage   |

## Output

**Get mode** (key only):

| Field | Type   | Description        |
|-------|--------|--------------------|
| key   | string | The storage key    |
| value | string | The stored value   |

```json
{"key":"wizard_step","value":"3"}
```

**Set mode** (key and value):

| Field | Type   | Description          |
|-------|--------|----------------------|
| key   | string | The storage key      |
| value | string | The value that was set |
| set   | bool   | Whether the value was set |

```json
{"key":"wizard_step","value":"4","set":true}
```

**Clear mode** (`--clear`):

| Field   | Type | Description                        |
|---------|------|------------------------------------|
| cleared | bool | Whether sessionStorage was cleared |

```json
{"cleared":true}
```

## Errors

| Condition            | Exit code | Stderr                        |
|----------------------|-----------|-------------------------------|
| Missing key argument | 1         | `error: key argument required`|
| Chrome not connected | 2         | `error: chrome not connected` |
| Operation timeout    | 3         | `error: timeout`              |

## Examples

Read a value:

```
hubcap session wizard_step
```

Set a value:

```
hubcap session wizard_step 4
```

Clear all sessionStorage:

```
hubcap session --clear
```

Clear session storage and reload the page (chaining):

```
hubcap session --clear && hubcap reload
```

## See also

- [storage](storage.md) - Get, set, or clear localStorage entries
- [cookies](cookies.md) - Manage browser cookies
- [clipboard](clipboard.md) - Read or write the system clipboard
