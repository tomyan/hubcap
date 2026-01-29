# hubcap cookies

Manage browser cookies.

## When to use

Use `cookies` to list, set, delete, or clear browser cookies. With no flags, it lists all cookies. Use `storage` for localStorage or `session` for sessionStorage.

## Usage

```
hubcap cookies [flags]
```

## Arguments

None.

## Flags

| Flag     | Type   | Default | Description                        |
|----------|--------|---------|------------------------------------|
| --set    | string | ""      | Cookie name=value to set           |
| --delete | string | ""      | Cookie name to delete              |
| --clear  | bool   | false   | Clear all cookies                  |
| --domain | string | ""      | Cookie domain for set/delete       |

## Output

**List mode** (no flags):

| Field    | Type   | Description           |
|----------|--------|-----------------------|
| (array)  | array  | Array of cookie objects |

```json
[{"name":"session_id","value":"abc123","domain":".example.com","path":"/","expires":1735689600,"httpOnly":true,"secure":true,"sameSite":"Lax"}]
```

**Set mode** (`--set`):

| Field  | Type   | Description            |
|--------|--------|------------------------|
| set    | bool   | Whether the cookie was set |
| name   | string | Cookie name            |
| value  | string | Cookie value           |
| domain | string | Cookie domain          |

```json
{"set":true,"name":"theme","value":"dark","domain":".example.com"}
```

**Delete mode** (`--delete`):

| Field   | Type   | Description              |
|---------|--------|--------------------------|
| deleted | bool   | Whether the cookie was deleted |
| name    | string | Cookie name              |
| domain  | string | Cookie domain            |

```json
{"deleted":true,"name":"session_id","domain":".example.com"}
```

**Clear mode** (`--clear`):

| Field   | Type | Description                  |
|---------|------|------------------------------|
| cleared | bool | Whether all cookies were cleared |

```json
{"cleared":true}
```

## Errors

| Condition            | Exit code | Stderr                        |
|----------------------|-----------|-------------------------------|
| Invalid --set format | 1         | `error: invalid cookie format, use name=value` |
| Chrome not connected | 2         | `error: connecting to Chrome: ...` |
| Timeout              | 3         | `error: timeout`              |

## Examples

List all cookies:

```
hubcap cookies
```

Set a cookie with a domain:

```
hubcap cookies --set theme=dark --domain .example.com
```

Delete a specific cookie:

```
hubcap cookies --delete session_id
```

Clear all cookies:

```
hubcap cookies --clear
```

Clear cookies before navigating (chaining):

```
hubcap cookies --clear && hubcap goto --wait https://example.com
```

## See also

- [storage](storage.md) - Get, set, or clear localStorage entries
- [session](session.md) - Get, set, or clear sessionStorage entries
- [clipboard](clipboard.md) - Read or write the system clipboard
