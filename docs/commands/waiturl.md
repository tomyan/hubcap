# hubcap waiturl

Wait for the page URL to match a pattern.

## When to use

Use `waiturl` to block until the current page URL matches a given pattern. Use after clicking links or submitting forms that trigger navigation.

## Usage

```
hubcap waiturl <pattern> [--timeout <duration>]
```

## Arguments

| Argument | Type   | Required | Description                  |
|----------|--------|----------|------------------------------|
| pattern  | string | Yes      | URL pattern to match         |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field   | Type   | Description             |
|---------|--------|-------------------------|
| pattern | string | The pattern matched     |
| url     | string | The URL that matched    |

```json
{"pattern":"/dashboard","url":"https://example.com/dashboard"}
```

## Errors

| Condition                        | Exit code | Stderr                                  |
|----------------------------------|-----------|------------------------------------------|
| Missing pattern argument         | 1         | `usage: hubcap waiturl <pattern> [--timeout <duration>]` |
| Chrome not connected             | 2         | `error: connecting to Chrome: ...`       |
| URL not matched within timeout   | 3         | `error: timeout`                         |

## Examples

Wait for a redirect to the dashboard:

```
hubcap waiturl '/dashboard'
```

Wait for a URL containing a query parameter:

```
hubcap waiturl 'status=success' --timeout 10s
```

Submit a login form and wait for redirect (chaining):

```
hubcap click '#login-btn' && hubcap waiturl '/dashboard' --timeout 15s
```

## See also

- [waitnav](waitnav.md) - Wait for any navigation event
- [waitload](waitload.md) - Wait for the page load event
- [url](url.md) - Get the current page URL
- [goto](goto.md) - Navigate to a URL
