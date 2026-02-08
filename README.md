# hubcap

A command-line interface for Chrome DevTools Protocol. Control headless and headed Chrome from your terminal, scripts, and CI pipelines.

Named after the spinning chrome hubcaps of classic cars — because this tool puts a shine on browser automation.

## Quick start

Start Chrome with remote debugging enabled:

```bash
# macOS
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222

# Linux
google-chrome --remote-debugging-port=9222

# Headless
google-chrome --headless --remote-debugging-port=9222
```

Then use hubcap:

```bash
# Navigate and interact
hubcap goto https://example.com
hubcap click '#login-button'
hubcap fill '#email' 'user@example.com'
hubcap type 'my-password'
hubcap press Enter

# Extract data
hubcap title
hubcap text '#main-content'
hubcap eval 'document.querySelectorAll("a").length'

# Capture output
hubcap screenshot --output page.png
hubcap pdf --output page.pdf

# Wait for things
hubcap wait '#results'
hubcap waittext 'Success'
hubcap waitidle
```

## Install

```bash
go install github.com/tomyan/hubcap/cmd/hubcap@latest
```

Or build from source:

```bash
git clone https://github.com/tomyan/hubcap.git
cd hubcap
go build -o hubcap ./cmd/hubcap
```

## How it works

Every command connects to Chrome via WebSocket, sends one or more Chrome DevTools Protocol messages, and prints a JSON result to stdout. Errors go to stderr. Exit codes tell you what happened:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Command error (bad args, element not found, JS error) |
| 2 | Connection failed (Chrome not running or wrong port) |
| 3 | Timeout |

This makes hubcap composable with standard Unix tools:

```bash
# Chain commands
hubcap goto https://example.com && hubcap screenshot --output shot.png

# Parse JSON output with jq
hubcap tabs | jq '.[].url'

# Use in scripts
TITLE=$(hubcap title | jq -r '.title')
echo "Page title is: $TITLE"

# Loop over elements
COUNT=$(hubcap count '.item' | jq '.count')
echo "Found $COUNT items"
```

## Global flags

Every command accepts these flags before the command name:

```
-port <n>        Chrome debug port (default: 9222, env: HUBCAP_PORT)
-host <s>        Chrome debug host (default: localhost, env: HUBCAP_HOST)
-timeout <d>     Command timeout (default: 10s)
-output <fmt>    Output format: json, ndjson, text (default: json)
-quiet           Suppress non-essential output
-target <id>     Target page by index (0-based) or target ID
```

Examples:

```bash
# Connect to Chrome on a different port
hubcap -port 9333 version

# Target the second tab
hubcap -target 1 title

# Target a specific tab by ID
hubcap -target "ABC123DEF456" click '#btn'

# Set a longer timeout for slow pages
hubcap -timeout 30s goto --wait https://slow-site.com

# Use environment variables
export HUBCAP_PORT=9333
hubcap tabs
```

## Configuration

Create a `.hubcaprc` file in your project directory or home directory to set defaults:

```json
{
  "port": 9333,
  "host": "localhost",
  "timeout": "30s",
  "output": "json"
}
```

Precedence: CLI flags > environment variables > `.hubcaprc` > built-in defaults.

## Common workflows

### Assertions and retry

```bash
# Assert page state
hubcap assert title "Dashboard"
hubcap assert exists '#user-menu'
hubcap assert text '#status' "Active"
hubcap assert count '.notification' 3

# Retry flaky checks
hubcap retry --attempts 5 --interval 2s assert text '#status' "Ready"
```

### Scripting with pipe

```bash
# Run commands from a file
hubcap pipe < test-script.txt

# Inline script
hubcap pipe <<'EOF'
goto https://example.com
wait '#content'
assert title "Example Domain"
screenshot --output page.png
EOF
```

### Interactive exploration

```bash
hubcap shell
hubcap> goto https://example.com
hubcap> title
hubcap> .output text
hubcap> text h1
hubcap> .quit
```

### Form submission

```bash
hubcap goto --wait https://example.com/login
hubcap fill '#username' 'admin'
hubcap fill '#password' 'secret'
hubcap click '#submit'
hubcap waitnav
hubcap title
```

### Scraping data

```bash
hubcap goto --wait https://news.example.com
hubcap eval 'JSON.stringify([...document.querySelectorAll("h2")].map(h => h.textContent))'
```

### Mobile testing

```bash
hubcap emulate "iPhone 12"
hubcap goto --wait https://example.com
hubcap screenshot --output mobile.png
hubcap tap '#menu-button'
hubcap swipe '#carousel' left
```

### Accessibility audit

```bash
hubcap goto --wait https://example.com
hubcap a11y | jq '.nodes[] | select(.role == "button")'
```

### Performance analysis

```bash
hubcap goto --wait https://example.com
hubcap metrics | jq '.metrics'
hubcap coverage
hubcap csscoverage
hubcap trace --duration 2s --output trace.json
```

### Network debugging

```bash
# Monitor all requests
hubcap network --duration 10s

# Wait for a specific API call
hubcap waitresponse '/api/data'

# Block resources
hubcap block '*.ads.js' '*.tracking.com'

# Simulate slow network
hubcap throttle slow3g
```

### Multi-tab workflows

```bash
# Open a new tab
hubcap new https://example.com

# List all tabs
hubcap tabs

# Work in specific tabs
hubcap -target 0 title
hubcap -target 1 title

# Close a tab
hubcap -target 1 close
```

### Memory debugging

```bash
hubcap goto --wait https://my-app.com
hubcap heapsnapshot --output before.json
# ... interact with app ...
hubcap heapsnapshot --output after.json
```

## Output format

All commands output JSON by default. Use `-output text` for plain text or `-output ndjson` for streaming newline-delimited JSON (used by monitoring commands like `console`, `network`, `errors`).

```bash
# JSON (default)
hubcap title
# {"title":"Example Domain"}

# Plain text
hubcap -output text title
# Example Domain

# Streaming NDJSON
hubcap -output ndjson console --duration 5s
# {"type":"log","text":"Hello"}
# {"type":"warn","text":"Deprecated API"}
```

## Command reference

See [docs/commands.md](docs/commands.md) for the full command directory, or individual command docs in the [docs/commands/](docs/commands/) folder.

There are 113 commands organized into these categories:

- **Browser & tabs** — version, tabs, new, close
- **Navigation** — goto, back, forward, reload, waitnav, waitload, waiturl
- **Page info** — title, url, info, source, meta, links, scripts, images, tables, forms, frames
- **DOM queries** — query, html, text, attr, value, count, visible, exists, bounds, styles, computed, layout, shadow, find, selection, caret
- **Click & input** — click, dblclick, rightclick, tripleclick, clickat, hover, tap, focus, fill, clear, type, press, select, check, uncheck, setvalue, upload, dispatch, drag, mouse
- **Touch gestures** — swipe, pinch
- **Scrolling** — scroll, scrollto, scrolltop, scrollbottom
- **Waiting** — wait, waittext, waitgone, waitfn, waitidle, waitrequest, waitresponse
- **Screenshots & export** — screenshot, pdf
- **Cookies & storage** — cookies, storage, session, clipboard
- **Network** — network, har, intercept, block, throttle, waitrequest, waitresponse, responsebody
- **Device emulation** — emulate, useragent, geolocation, offline, media, viewport, permission
- **Monitoring** — console, errors, network, har
- **Analysis** — metrics, a11y, coverage, csscoverage, stylesheets, listeners, domsnapshot
- **Profiling** — heapsnapshot, trace
- **Assert** — assert (text, title, url, exists, visible, count)
- **Utility** — retry, pipe, shell, record, help
- **Advanced** — eval, evalframe, run, raw, dialog, highlight

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...
```

## License

MIT
