# hubcap raw

Send a raw Chrome DevTools Protocol command.

## When to use

Send raw Chrome DevTools Protocol commands for protocol methods not covered by dedicated hubcap commands. This is an escape hatch for advanced or uncommon protocol interactions. Use `--browser` for browser-level commands like `Target.getTargets` or `Browser.getVersion`.

## Usage

```
hubcap raw [--browser] <method> [params-json]
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `method` | string | Yes | protocol method name (e.g. `Page.reload`, `DOM.getDocument`) |
| `params-json` | string | No | JSON string of method parameters |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--browser` | bool | `false` | Send command at browser level instead of page level |

## Output

The raw protocol result JSON, exactly as returned by the protocol.

| Field | Type | Description |
|-------|------|-------------|
| *(varies)* | object | The protocol method response, structure depends on the method called |

```json
{"frameTree":{"frame":{"id":"main","url":"https://example.com"}}}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Missing method argument | 1 | `usage: hubcap raw [--browser] <method> [params-json]` |
| Protocol error | 1 | `error: <message>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout waiting for response | 3 | `error: timeout` |

## Examples

Reload the page:

```
hubcap raw Page.reload
```

Get the document root node:

```
hubcap raw DOM.getDocument '{"depth": 1}'
```

Target the browser to list all open targets:

```
hubcap raw --browser Target.getTargets
```

Get browser version info:

```
hubcap raw --browser Browser.getVersion
```

Enable a protocol domain and pipe the result to jq for filtering:

```
hubcap raw Network.enable && hubcap raw Network.getResponseBody '{"requestId":"1234"}' | jq '.body'
```

## See also

- [eval](eval.md) - Evaluate JavaScript in the page without raw protocol
- [version](version.md) - Print browser version information
