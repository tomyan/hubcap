# hubcap waitresponse

Wait for a network response matching a URL pattern.

## When to use

Use `waitresponse` to block until a network response matching a URL pattern is detected. Use `waitrequest` to wait for the request instead. Use `responsebody` to get the response content.

## Usage

```
hubcap waitresponse <pattern> [--timeout <duration>]
```

## Arguments

| Argument | Type   | Required | Description                            |
|----------|--------|----------|----------------------------------------|
| pattern  | string | Yes      | URL pattern to match against responses |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field     | Type   | Description                          |
|-----------|--------|--------------------------------------|
| found     | bool   | Whether a matching response was detected |
| url       | string | The full URL of the matched response |
| status    | int    | HTTP status code                     |
| mimeType  | string | MIME type of the response            |
| requestId | string | Protocol request identifier           |

```json
{"found":true,"url":"https://api.example.com/users","status":200,"mimeType":"application/json","requestId":"1234.5"}
```

## Errors

| Condition                           | Exit code | Stderr                                     |
|-------------------------------------|-----------|---------------------------------------------|
| No matching response within timeout | 3         | `error: timeout waiting for response`       |
| Chrome not connected                | 2         | `error: chrome not connected`               |
| Missing pattern argument            | 1         | `error: pattern argument required`          |

## Examples

Wait for a successful API response:

```
hubcap waitresponse '/api/users'
```

Wait for a specific resource with a timeout:

```
hubcap waitresponse '/data.json' --timeout 10s
```

Click a button and capture the API response (chaining):

```
hubcap click '#save' && hubcap waitresponse '/api/save' | jq '.status'
```

## See also

- [waitrequest](waitrequest.md) - Wait for a network request matching a URL pattern
- [responsebody](responsebody.md) - Get the body of a network response
- [network](network.md) - Monitor network activity
