# hubcap har - Capture network activity in HAR format

## When to use

Capture network activity in standard HTTP Archive (HAR) format for analysis, replay, or import into tools like Chrome DevTools. Use `network` for real-time NDJSON streaming instead.

## Usage

```
hubcap har [--duration <duration>]
```

## Arguments

None.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--duration` | duration | `5s` | How long to capture |

## Output

A single HAR format JSON object written to stdout conforming to the HTTP Archive 1.2 specification.

```json
{
  "log": {
    "version": "1.2",
    "creator": {
      "name": "hubcap",
      "version": "1.0.0"
    },
    "entries": [
      {
        "startedDateTime": "2025-01-15T10:00:00.000Z",
        "request": {
          "method": "GET",
          "url": "https://example.com/api/data",
          "headers": []
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "headers": [],
          "content": {
            "size": 1234,
            "mimeType": "application/json"
          }
        },
        "time": 150
      }
    ]
  }
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Duration parse failure | 1 | `error: invalid duration "<value>"` |
| Timeout exceeded | 3 | `error: timeout` |

## Examples

Capture 5 seconds of network activity (default duration):

```bash
hubcap har
```

Capture 30 seconds and save to a file:

```bash
hubcap har --duration 30s > trace.har
```

Capture and count the number of requests:

```bash
hubcap har --duration 10s | jq '.log.entries | length'
```

Navigate to a page and capture the resulting network activity:

```bash
hubcap goto "https://example.com" && hubcap har --duration 10s > page-load.har
```

## See also

- [network](network.md) - Stream network requests and responses as NDJSON
- [responsebody](responsebody.md) - Get the response body for a captured request
- [intercept](intercept.md) - Intercept and modify network requests or responses
