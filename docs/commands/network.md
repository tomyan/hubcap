# hubcap network - Stream network requests and responses

## When to use

Monitor network requests and responses in real time as NDJSON. Use `har` for standard HAR format output suitable for analysis tools. Use `waitrequest` or `waitresponse` to block until a specific network event occurs.

## Usage

```
hubcap network [--duration <duration>]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--duration` | duration | `0` | How long to capture; 0 = until interrupted |

## Output

NDJSON stream written to stdout. Each line is a JSON object representing either a request or a response.

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `"request"` or `"response"` |
| `requestId` | string | Unique identifier for the request/response pair |
| `url` | string | Request URL |
| `method` | string | HTTP method (request lines only) |
| `status` | int | HTTP status code (response lines only) |
| `mimeType` | string | Response MIME type (response lines only) |

Request line:

```json
{"type":"request","requestId":"1000.1","method":"GET","url":"https://example.com/api/data"}
```

Response line:

```json
{"type":"response","requestId":"1000.1","status":200,"mimeType":"application/json"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not reachable | 2 | `error: cannot connect to Chrome` |
| Duration parse failure | 1 | `error: invalid duration "<value>"` |

## Examples

Stream all network activity until Ctrl-C:

```bash
hubcap network
```

Capture network activity for 10 seconds:

```bash
hubcap network --duration 10s
```

Filter for API requests using jq:

```bash
hubcap network --duration 30s | jq 'select(.type == "request" and (.url | contains("/api/")))'
```

Get the response body for a captured request:

```bash
hubcap network --duration 5s | jq -r 'select(.type == "response" and .status == 200) | .requestId' | head -1 | xargs hubcap responsebody
```

## See also

- [har](har.md) - Capture network activity in HAR format
- [responsebody](responsebody.md) - Get the response body for a captured request
- [waitrequest](waitrequest.md) - Wait for a specific network request
- [waitresponse](waitresponse.md) - Wait for a specific network response
- [block](block.md) - Block network requests by URL pattern
- [intercept](intercept.md) - Intercept and modify network requests or responses
