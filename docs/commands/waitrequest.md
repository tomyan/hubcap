# hubcap waitrequest

Wait for a network request matching a URL pattern.

## When to use

Use `waitrequest` to block until a network request matching a URL pattern is detected. Use `waitresponse` to wait for the response instead.

## Usage

```
hubcap waitrequest <pattern> [--timeout <duration>]
```

## Arguments

| Argument | Type   | Required | Description                          |
|----------|--------|----------|--------------------------------------|
| pattern  | string | Yes      | URL pattern to match against requests |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field     | Type   | Description                        |
|-----------|--------|------------------------------------|
| found     | bool   | Whether a matching request was detected |
| url       | string | The full URL of the matched request |
| method    | string | HTTP method of the request         |
| requestId | string | Protocol request identifier         |

```json
{"found":true,"url":"https://api.example.com/users","method":"GET","requestId":"1234.5"}
```

## Errors

| Condition                          | Exit code | Stderr                                    |
|------------------------------------|-----------|-------------------------------------------|
| Missing pattern argument           | 1         | `usage: hubcap waitrequest <pattern> [--timeout <duration>]` |
| Chrome not connected               | 2         | `error: connecting to Chrome: ...`        |
| No matching request within timeout | 3         | `error: timeout`                          |

## Examples

Wait for an API call:

```
hubcap waitrequest '/api/users'
```

Wait for a POST request with a timeout:

```
hubcap waitrequest '/api/submit' --timeout 10s
```

Click a button and wait for the resulting API call (chaining):

```
hubcap click '#load-more' && hubcap waitrequest '/api/items'
```

## See also

- [waitresponse](waitresponse.md) - Wait for a network response matching a URL pattern
- [network](network.md) - Monitor network activity
- [intercept](intercept.md) - Intercept and modify network requests
