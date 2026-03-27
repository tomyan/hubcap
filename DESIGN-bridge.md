# Bridge command design

Persistent bidirectional message channel between a hubcap CLI process and client-side JavaScript running in a browser tab.

## Motivation

When integrating with web apps that have client-side JavaScript APIs, you sometimes need two-way communication between a local process and the page. For example, syncing data between a web app and a local database, or having an LLM interact with a page's API at low latency. Browser security measures (CORS, CSP, mixed content) can block direct server connections from the page, but injecting JS via CDP bypasses these restrictions.

## Command interface

```bash
# Inline JS
hubcap bridge --target <id> '<js body>'

# From file
hubcap bridge --target <id> --file script.js
```

The JS body receives `messages` (async iterator) and `send` (function):

```js
for await (const msg of messages) {
  const result = await somePageAPI(msg);
  send(result);
}
```

## Protocol

### stdout (LDJSON, one JSON object per line)

```jsonl
{"type":"ready"}
{"type":"message","data":<value>}
{"type":"error","error":"<message>","stack":"..."}
{"type":"closed","reason":"<reason>"}
```

- `ready` — bridge established, JS is running
- `message` — value passed to `send()` from JS
- `error` — uncaught error in user JS or send() rejection
- `closed` — bridge ended (tab closed, navigation, explicit close, keepalive timeout)

### stdin (LDJSON)

```jsonl
{"data":<value>}
{"type":"close"}
```

- `data` — delivered to the async iterator in JS
- `close` — graceful shutdown from CLI side

### stderr

Diagnostics and errors (connection failures, protocol errors).

## Internal mechanism

### Instance isolation

Each bridge gets a random ID (e.g. `__hubcap_bridge_a7f3b2`) to namespace all injected globals. Multiple bridges can coexist in the same tab.

### JS→CLI (outbound messages)

`Runtime.addBinding` registers a function `<id>_send`. When the user calls `send(value)`, it serialises to JSON and calls the binding. The `Runtime.bindingCalled` CDP event delivers it to hubcap, which writes a `{"type":"message","data":...}` line to stdout.

### CLI→JS (inbound messages)

An injected script maintains an async iterator backed by a promise queue. When hubcap reads a line from stdin, it calls `Runtime.evaluate` to invoke `<id>_push(msg)`, which resolves the next pending promise in the iterator.

The async iterator follows the standard pattern:
- A queue of `{resolve, reject}` pairs (pending reads)
- A buffer of messages (if push happens before next read)
- `push(msg)` either resolves a pending read or buffers the message
- `close()` resolves all pending reads with `{done: true}`

### Keepalive

Hubcap sends a heartbeat via `Runtime.evaluate` every 2 seconds, calling `<id>_heartbeat()`. The client-side wrapper tracks the last heartbeat timestamp. If 2 consecutive heartbeats are missed (>6 seconds), the iterator closes and cleanup runs.

On the hubcap side:
- `Target.detachedFromTarget` — tab closed
- `Inspector.targetCrashed` — tab crashed
- `Page.frameNavigated` — top-level navigation destroyed JS context

### Error handling

- Uncaught errors in user JS: caught by a try/catch wrapper around the user's script body. Serialised to stdout as `{"type":"error",...}`.
- `send()` errors: if the CDP binding call fails, the promise rejects.
- CDP disconnection: hubcap writes `{"type":"closed","reason":"..."}` and exits.
- stdin EOF: hubcap sends close signal to JS, waits briefly for cleanup, then exits.

## Iteration plan

### 1. One-way outbound: JS → stdout

The thinnest slice. Inject JS that calls `send()`, messages appear as LDJSON on stdout.

- New `bridge` command that accepts inline JS
- `Runtime.addBinding` for send
- `Runtime.bindingCalled` event listener writes to stdout
- `{"type":"ready"}` emitted when JS starts
- Command stays alive until ctrl-c or tab close
- Tests: send a message from JS, verify it appears on stdout

### 2. One-way inbound: stdin → JS

Add the async iterator. Messages from stdin get delivered to JS.

- Inject the async iterator machinery (promise queue)
- Read stdin line by line, call `<id>_push()` via `Runtime.evaluate`
- Tests: write to stdin, verify JS receives via iterator

### 3. Bidirectional + close

Wire both directions together. Add graceful close.

- `{"type":"close"}` from stdin triggers iterator close
- stdin EOF triggers close
- User JS exiting (iterator loop ends) triggers `{"type":"closed"}`
- Tests: round-trip message flow, close from both sides

### 4. Keepalive

Add heartbeat mechanism.

- Hubcap sends heartbeat every 2s
- JS detects missed heartbeats and closes iterator
- Tests: simulate hubcap death (stop heartbeats), verify JS exits

### 5. Error handling

Robust error reporting.

- Wrap user JS in try/catch, emit `{"type":"error",...}`
- Handle CDP disconnection, tab crash, navigation
- Tests: JS throws, tab navigates, tab closes

### 6. File input + instance isolation

- `--file` flag to load JS from file
- Random bridge ID namespacing
- Tests: two bridges in same tab, file-based script

### 7. Docs + skill update

- `docs/commands/bridge.md`
- Update skill SKILL.md with bridge section
- Update docs site
