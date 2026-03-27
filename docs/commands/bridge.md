# hubcap bridge

Persistent bidirectional message channel between the CLI and client-side JavaScript.

## When to use

Use `bridge` when you need two-way communication with JavaScript running in a page. The command stays alive, streaming messages as LDJSON over stdio. This is useful for syncing data with a web app's client-side API, driving interactive page workflows, or building real-time integrations.

## Usage

```
hubcap bridge <script>
hubcap bridge --file <file>
```

The script receives two variables in scope:

- `messages` — async iterator of messages sent from stdin
- `send(data)` — function to send a message to stdout

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `script` | string | Yes (or `--file`) | JavaScript to run in the page context |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--file` | string | | Load JavaScript from a file instead of inline |

## Protocol

### stdout (LDJSON)

Each line is a JSON object with a `type` field:

| Type | Fields | Description |
|------|--------|-------------|
| `ready` | | Bridge established, JS is running |
| `message` | `data` | Value passed to `send()` from JS |
| `error` | `error` | Uncaught JS error or send failure |
| `closed` | `data` | Bridge ended (script finished, tab closed, keepalive timeout) |

### stdin (LDJSON)

| Type | Fields | Description |
|------|--------|-------------|
| (default) | `data` | Delivered to the `messages` async iterator |
| `close` | `type` | Graceful shutdown |

### stderr

Diagnostics and connection errors.

## Keepalive

The bridge sends a heartbeat every 2 seconds. If the JS side misses heartbeats for 6 seconds (e.g. the hubcap process was killed), the async iterator closes automatically.

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Script injection failed | 1 | `error: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Send a message from JS:

```bash
hubcap bridge 'send({title: document.title})'
```

Echo messages back (stdin → JS → stdout):

```bash
echo '{"data":{"n":7}}' | hubcap bridge '
  for await (const msg of messages) {
    send({doubled: msg.n * 2});
    break;
  }
'
```

Sync with a page API:

```bash
hubcap bridge --file sync.js --target "$TAB"
```

Where `sync.js` might be:

```js
for await (const msg of messages) {
  if (msg.action === "get") {
    const data = await window.appAPI.getData(msg.key);
    send({key: msg.key, value: data});
  } else if (msg.action === "set") {
    await window.appAPI.setData(msg.key, msg.value);
    send({key: msg.key, ok: true});
  }
}
```

## See also

- [eval](eval.md) - One-shot JavaScript evaluation
- [run](run.md) - Run a JavaScript file (one-shot)
- [console](console.md) - Capture console messages (one-way)
