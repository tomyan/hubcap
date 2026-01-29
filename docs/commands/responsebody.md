# hubcap responsebody - Get the response body for a network request

## When to use

Get the response body for a previously captured network request. Use the `requestId` from `network` or `har` output to identify which response to retrieve.

## Usage

```
hubcap responsebody <requestId>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `requestId` | string | Yes | Request identifier from `network` or `har` output |

## Flags

None.

## Output

The response body content object.

| Field | Type | Description |
|-------|------|-------------|
| `body` | string | Response body content |
| `base64Encoded` | bool | Whether the body is base64-encoded |

```json
{"body":"{\"status\":\"ok\",\"data\":[1,2,3]}","base64Encoded":false}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not reachable | 2 | `error: cannot connect to Chrome` |
| Missing requestId argument | 1 | `error: requestId argument is required` |
| Request not found | 1 | `error: request "<requestId>" not found` |

## Examples

Get the response body for a specific request ID:

```bash
hubcap responsebody "1000.1"
```

Extract the decoded body content:

```bash
hubcap responsebody "1000.1" | jq -r '.body'
```

Capture network traffic and then retrieve the body for the first JSON response:

```bash
REQUEST_ID=$(hubcap network --duration 5s | jq -r 'select(.type == "response" and .mimeType == "application/json") | .requestId' | head -1)
hubcap responsebody "$REQUEST_ID" | jq -r '.body' | jq .
```

## See also

- [network](network.md) - Stream network requests and responses
- [har](har.md) - Capture network activity in HAR format
- [intercept](intercept.md) - Intercept and modify network requests or responses
- [waitresponse](waitresponse.md) - Wait for a specific network response
