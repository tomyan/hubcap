# hubcap waitnav

Wait for any navigation event to occur.

## When to use

Use `waitnav` to block until any navigation event is detected. Use `waiturl` to wait for a specific URL pattern. Use `waitload` to wait specifically for the page load event.

## Usage

```
hubcap waitnav [--timeout <duration>]
```

## Arguments

None.

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field     | Type | Description                          |
|-----------|------|--------------------------------------|
| navigated | bool | Whether a navigation event occurred  |

```json
{"navigated":true}
```

## Errors

| Condition                          | Exit code | Stderr                                    |
|------------------------------------|-----------|-------------------------------------------|
| Chrome not connected               | 2         | `error: connecting to Chrome: ...`        |
| No navigation within timeout       | 3         | `error: timeout`                          |

## Examples

Wait for navigation after clicking a link:

```
hubcap click '#next-page' && hubcap waitnav
```

Wait with a short timeout:

```
hubcap waitnav --timeout 5s
```

Click a link and wait for the navigation to complete (chaining):

```
hubcap click 'a.continue' && hubcap waitnav && hubcap title
```

## See also

- [waitload](waitload.md) - Wait for the page load event
- [waiturl](waiturl.md) - Wait for the URL to match a pattern
- [goto](goto.md) - Navigate to a URL
- [waitidle](waitidle.md) - Wait for network idle
