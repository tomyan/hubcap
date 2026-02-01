# hubcap reload

Reload the current page.

## When to use

Use `reload` to reload the current page. Use `--bypass-cache` to force fresh resources from the server. Use `goto` to navigate to a different URL.

## Usage

```
hubcap reload [--bypass-cache]
```

## Arguments

None.

## Flags

| Flag           | Type | Default | Description              |
|----------------|------|---------|--------------------------|
| --bypass-cache | bool | false   | Bypass browser cache     |

## Output

| Field       | Type | Description                              |
|-------------|------|------------------------------------------|
| reloaded    | bool | Whether the page reload was triggered    |
| ignoreCache | bool | Whether the cache was bypassed           |

```json
{"reloaded":true,"ignoreCache":false}
```

## Errors

| Condition            | Exit code | Stderr                        |
|----------------------|-----------|-------------------------------|
| Chrome not connected | 2         | `error: connecting to Chrome: ...` |
| Timeout              | 3         | `error: timeout`                   |

## Examples

Reload the current page:

```
hubcap reload
```

Reload while bypassing the cache:

```
hubcap reload --bypass-cache
```

Force-reload and wait for the page to finish loading (chaining):

```
hubcap reload --bypass-cache && hubcap waitload
```

## See also

- [goto](goto.md) - Navigate to a URL
- [back](back.md) - Navigate back in history
- [forward](forward.md) - Navigate forward in history
- [waitload](waitload.md) - Wait for the page load event
