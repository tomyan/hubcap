# CDP CLI Design

A Go CLI for Chrome DevTools Protocol interaction, designed for use by Claude agent skills.

## Inspiration

- [myers/cdp-cli](https://github.com/myers/cdp-cli) - NDJSON output, token-efficient console, object inspection
- [browser-debugger-cli](https://github.com/szymdzum/browser-debugger-cli) - Self-discovery, progressive disclosure
- [chrome-devtools-mcp](https://github.com/ChromeDevTools/chrome-devtools-mcp) - High-level automation patterns
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/) - Official protocol reference (65+ domains)
- [chromedp/examples](https://github.com/chromedp/examples) - Go CDP usage patterns
- [Puppeteer](https://pptr.dev/) / [Playwright](https://playwright.dev/) - High-level automation APIs built on CDP

## Research Findings

### CDP vs Higher-Level Tools

| Aspect | Raw CDP | Puppeteer/Playwright |
|--------|---------|---------------------|
| Speed | Faster (15-20% in benchmarks) | Slower due to JS injection |
| Message size | ~11KB for typical task | ~326KB (Playwright) |
| Control | Full protocol access | Abstracted, some features hidden |
| Complexity | Higher | Lower |

For an agent CLI, raw CDP access is preferred - agents can compose low-level operations and don't need the "developer convenience" abstractions.

### Key CDP Capabilities to Expose

**Network Analysis**
- HAR file generation from Network domain events
- Request interception via Fetch domain (not deprecated Network.setRequestInterception)
- Response body modification with `Fetch.fulfillRequest`

**Coverage Analysis**
- CSS/JS coverage via Profiler domain
- Identifies unused code (~30% typical savings)
- Export in JSON or LCOV format

**Core Web Vitals**
- LCP from `largestContentfulPaint::Candidate` trace events
- CLS from `LayoutShift` events (with `had_recent_input: false`)
- FCP from `firstContentfulPaint` trace event
- TTFB from `ResourceReceiveResponse` - `ResourceSendRequest` timestamps

**Memory Profiling**
- Heap snapshots via HeapProfiler domain
- Allocation timeline tracking
- Leak detection (detached DOM nodes, closures)

**Storage Management**
- Cookies (all domains, HttpOnly accessible via CDP)
- LocalStorage / SessionStorage via DOMStorage domain
- IndexedDB via IndexedDB domain
- Cache Storage (Service Worker caches)

**Device Emulation Presets**
- Mobile S (320px), Mobile M (375px), Mobile L (425px)
- Tablet (768px), Laptop (1024px), Laptop L (1440px), 4K (2560px)
- Custom device pixel ratios, touch emulation, user agents

**Source Map Resolution**
- Parse `//# sourceMappingURL` comments
- Resolve and fetch source maps
- Map minified positions to original source

**PWA/Service Worker**
- Service worker registration and lifecycle
- Cache storage inspection
- Background sync/fetch events
- Push notification debugging

**Accessibility Testing**
- Accessibility tree via Accessibility domain
- Integration point for axe-core rules
- Vision deficiency emulation

## Extension Replacement

This CLI can replace the need for many Chrome extensions, providing programmatic access to the same functionality. Key extension categories and CLI equivalents:

| Extension Category | Popular Extensions | CLI Equivalent |
|-------------------|-------------------|----------------|
| **JSON Viewer** | JSON Formatter, JSONView | `cdp network getResponseBody` + jq |
| **Header Modifier** | ModHeader, Requestly | `cdp intercept`, `cdp network headers` |
| **Storage Editor** | Storage Explorer, Easy Local Storage | `cdp storage local/session/indexeddb` |
| **Cookie Manager** | EditThisCookie, Cookie Editor | `cdp storage cookies` |
| **Screenshot** | GoFullPage, Full Page Screen Capture | `cdp screenshot --full-page` |
| **Form Filler** | Fake Filler, Testofill | `cdp fill-form --data` |
| **Responsive Tester** | Responsive Viewer, Viewport Resizer | `cdp emulate device/viewport` |
| **Selector Finder** | SelectorsHub, ChroPath | `cdp query`, `cdp a11y query` |
| **Performance** | Lighthouse, PageSpeed Insights | `cdp vitals`, `cdp trace`, `cdp coverage` |
| **React/Vue DevTools** | React Developer Tools, Vue DevTools | `cdp eval` + framework-specific queries |
| **Network Inspector** | HTTP-TRACKER, Postman Interceptor | `cdp watch requests`, `cdp har capture` |
| **Accessibility** | axe DevTools, WAVE | `cdp a11y audit`, `cdp a11y tree` |

### Advantages Over Extensions

1. **Scriptable** - Compose operations in shell scripts, CI pipelines
2. **No browser overhead** - Extensions can slow down pages
3. **Security** - No third-party extension code running in browser context
4. **Reproducible** - Same commands produce same results
5. **Agent-friendly** - JSON output parseable by LLMs
6. **No installation per-profile** - CLI works with any Chrome instance

### Additional Features from Extension Research

**Selector Generation** (from SelectorsHub, ChroPath):
```bash
# Generate selector for element at coordinates
cdp selector at --x 100 --y 200

# Generate multiple selector types for element
cdp selector for "<approximate-selector>" [--types xpath,css,playwright]

# Find unique selector for element
cdp selector unique --nodeId <id>

# Validate selector
cdp selector test "<selector>" [--count]
```

**Form Filling Profiles** (from Fake Filler, Testofill):
```bash
# Generate random data for form fields
cdp fill-form --random [--locale en-US]

# Save form state as reusable profile
cdp form save --name "login-test" --output profile.json

# Apply saved profile
cdp form load --profile profile.json

# Record form interactions
cdp form record --output interactions.json
```

**Responsive Testing** (from Responsive Viewer, Viewport Resizer):
```bash
# Capture screenshots at multiple viewports
cdp responsive capture --devices "iPhone 14,iPad Pro,Desktop" --output ./screenshots/

# Test at all breakpoints
cdp responsive breakpoints [--css-file styles.css]

# Compare rendering across viewports
cdp responsive diff --devices "mobile,desktop" --output diff.html
```

## Goals

1. **Full low-level CDP access** - All 65+ CDP domains/methods exposed as CLI commands
2. **Agent-friendly output** - JSON/NDJSON output, consistent error format, predictable behavior
3. **Self-discovery** - CLI can describe itself without external documentation
4. **Token efficiency** - Progressive verbosity, minimal default output
5. **High-level abstractions** - Composed operations that reduce context usage and round-trips
6. **Local queryable state** - SQLite sync for efficient resource querying without repeated CDP calls

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer (cobra)                        │
├─────────────────────────────────────────────────────────────────┤
│  Low-Level Commands          │  High-Level Commands             │
│  (1:1 CDP mapping)           │  (composed operations)           │
│                              │                                   │
│  cdp network enable          │  sync resources                  │
│  cdp page navigate           │  snapshot dom                    │
│  cdp dom getDocument         │  watch network                   │
│  cdp runtime evaluate        │  extract styles                  │
└──────────────┬───────────────┴──────────────┬────────────────────┘
               │                              │
               ▼                              ▼
┌─────────────────────────────┐  ┌─────────────────────────────────┐
│      CDP Client Layer       │  │       SQLite Storage Layer      │
│      (chromedp/cdproto)     │  │       (modernc.org/sqlite)      │
└──────────────┬──────────────┘  └──────────────┬──────────────────┘
               │                                │
               ▼                                ▼
┌─────────────────────────────┐  ┌─────────────────────────────────┐
│   Chrome Debug Port (:9222) │  │         Local .db File          │
└─────────────────────────────┘  └─────────────────────────────────┘
```

## Connection Management

```bash
# Connect to existing Chrome instance
cdp --port 9222 <command>

# Or use environment variable
CDP_PORT=9222 cdp <command>

# Connect via WebSocket URL directly
cdp --ws "ws://localhost:9222/devtools/browser/..." <command>

# Connection info stored for session reuse
cdp connect --port 9222 --save  # Saves to ~/.cdp/connection.json
```

### Launching Chrome

Helper to launch Chrome with remote debugging enabled:

```bash
# Launch Chrome with debugging
cdp launch [--port 9222] [--headless] [--user-data-dir <path>]

# Launch with specific Chrome channel
cdp launch --channel stable|beta|canary|dev

# Launch with custom flags
cdp launch --chrome-flags="--ignore-certificate-errors,--disable-web-security"

# Launch in temporary profile (clean state, auto-cleanup)
cdp launch --temp-profile

# Just print the launch command (don't execute)
cdp launch --dry-run
```

Platform-specific Chrome paths are auto-detected:
- **macOS**: `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
- **Linux**: `google-chrome` or `chromium-browser`
- **Windows**: `C:\Program Files\Google\Chrome\Application\chrome.exe`

## Self-Discovery

The CLI should be fully discoverable without external documentation.

```bash
# List all available domains
cdp --list
cdp domains

# List methods within a domain
cdp network --list
cdp dom --list

# Describe a specific method with full schema and examples
cdp network getResponseBody --describe
cdp dom querySelector --describe

# Search across all domains for methods/events
cdp --search cookie        # Find all cookie-related methods
cdp --search "request"     # Find request handling methods
```

Discovery output includes:
- Method signature with all parameters
- Parameter types and whether required/optional
- Return value schema
- Example invocations
- Related events

---

## Output Format

All commands support consistent output formatting:

```bash
cdp --output json <command>      # Single JSON object (default)
cdp --output ndjson <command>    # Newline-delimited JSON (streaming)
cdp --output text <command>      # Human-readable (for debugging)
```

### Standard Response Envelope

```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

### Error Format

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "CDP_TIMEOUT",
    "message": "Command timed out after 30s",
    "domain": "Network",
    "method": "getResponseBody",
    "context": { "requestId": "123.45" }
  }
}
```

### Semantic Exit Codes

```
0   Success
1   General error
2   Connection failed
3   Timeout
4   Invalid arguments
5   CDP protocol error
6   Target not found
7   Element not found
```

### Streaming (NDJSON)

For commands that produce multiple events:

```bash
cdp network watch --output ndjson
```

```
{"event":"requestWillBeSent","data":{"requestId":"1","url":"..."}}
{"event":"responseReceived","data":{"requestId":"1","status":200}}
{"event":"loadingFinished","data":{"requestId":"1"}}
```

### Token Efficiency Options

Minimize output for LLM context management:

```bash
# Console with progressive verbosity
cdp console list                           # Bare message strings only (default)
cdp console list --with-level              # Add log level
cdp console list --with-timestamp          # Add timestamps
cdp console list --with-source             # Add source file:line
cdp console list --verbose                 # Full structured output

# Object inspection control
cdp eval "window.config" --inspect         # Expand nested objects
cdp eval "largeArray" --inspect --depth 2  # Limit expansion depth
cdp eval "data" --no-truncate              # Don't truncate large values

# Limit results
cdp console list --tail 10                 # Last N messages
cdp network list --limit 50                # First N requests
```

---

## Low-Level Commands

Direct 1:1 mapping to CDP domains and methods. Auto-generated from CDP protocol definitions.

### Command Structure

```
cdp <domain> <method> [flags]
```

### Core Domains

#### Browser

```bash
cdp browser getVersion
cdp browser getWindowForTarget --targetId <id>
cdp browser close
```

#### Page

```bash
cdp page enable
cdp page disable
cdp page navigate --url <url>
cdp page reload [--ignoreCache]
cdp page getFrameTree
cdp page captureScreenshot [--format png|jpeg|webp] [--quality 0-100]
cdp page printToPDF [--landscape] [--printBackground]
cdp page getResourceTree
cdp page getResourceContent --frameId <id> --url <url>
```

#### DOM

```bash
cdp dom enable
cdp dom disable
cdp dom getDocument [--depth <n>] [--pierce]
cdp dom querySelector --nodeId <id> --selector <sel>
cdp dom querySelectorAll --nodeId <id> --selector <sel>
cdp dom getOuterHTML --nodeId <id>
cdp dom setOuterHTML --nodeId <id> --outerHTML <html>
cdp dom getAttributes --nodeId <id>
cdp dom setAttributeValue --nodeId <id> --name <n> --value <v>
cdp dom removeNode --nodeId <id>
cdp dom requestNode --objectId <id>
cdp dom describeNode --nodeId <id> [--depth <n>]
cdp dom getBoxModel --nodeId <id>
```

#### CSS

```bash
cdp css enable
cdp css disable
cdp css getMatchedStylesForNode --nodeId <id>
cdp css getComputedStyleForNode --nodeId <id>
cdp css getInlineStylesForNode --nodeId <id>
cdp css getStyleSheetText --styleSheetId <id>
cdp css setStyleSheetText --styleSheetId <id> --text <css>
cdp css getAllStyleSheets
```

#### Network

```bash
cdp network enable
cdp network disable
cdp network setCacheDisabled --cacheDisabled
cdp network setExtraHTTPHeaders --headers <json>
cdp network getResponseBody --requestId <id>
cdp network getCookies [--urls <urls>]
cdp network setCookie --name <n> --value <v> --domain <d>
cdp network deleteCookies --name <n> --domain <d>
cdp network clearBrowserCache
cdp network clearBrowserCookies
cdp network setUserAgentOverride --userAgent <ua>
cdp network emulateNetworkConditions --offline --latency <ms> --downloadThroughput <bps>
```

#### Fetch (Request Interception)

```bash
cdp fetch enable [--patterns <json>]
cdp fetch disable
cdp fetch continueRequest --requestId <id> [--url <url>] [--headers <json>]
cdp fetch fulfillRequest --requestId <id> --responseCode <n> --body <base64>
cdp fetch failRequest --requestId <id> --errorReason <reason>
cdp fetch getResponseBody --requestId <id>
```

#### Runtime

```bash
cdp runtime enable
cdp runtime disable
cdp runtime evaluate --expression <js> [--awaitPromise] [--returnByValue]
cdp runtime callFunctionOn --objectId <id> --functionDeclaration <js> [--arguments <json>]
cdp runtime getProperties --objectId <id> [--ownProperties]
cdp runtime releaseObject --objectId <id>
cdp runtime releaseObjectGroup --objectGroup <name>
```

#### Debugger

```bash
cdp debugger enable
cdp debugger disable
cdp debugger getScriptSource --scriptId <id>
cdp debugger setBreakpoint --location <json>
cdp debugger removeBreakpoint --breakpointId <id>
cdp debugger pause
cdp debugger resume
cdp debugger stepOver
cdp debugger stepInto
cdp debugger stepOut
cdp debugger evaluateOnCallFrame --callFrameId <id> --expression <js>
```

#### Target

```bash
cdp target getTargets
cdp target createTarget --url <url>
cdp target closeTarget --targetId <id>
cdp target attachToTarget --targetId <id>
cdp target detachFromTarget --sessionId <id>
cdp target activateTarget --targetId <id>
```

#### Emulation

```bash
cdp emulation setDeviceMetricsOverride --width <w> --height <h> --deviceScaleFactor <n> --mobile
cdp emulation clearDeviceMetricsOverride
cdp emulation setGeolocationOverride --latitude <lat> --longitude <lng>
cdp emulation setTimezoneOverride --timezoneId <tz>
cdp emulation setUserAgentOverride --userAgent <ua>
cdp emulation setTouchEmulationEnabled --enabled
```

#### Input

```bash
cdp input dispatchMouseEvent --type <type> --x <x> --y <y> [--button left|middle|right]
cdp input dispatchKeyEvent --type <type> --key <key> [--modifiers <n>]
cdp input insertText --text <text>
cdp input dispatchTouchEvent --type <type> --touchPoints <json>
```

#### Accessibility

```bash
cdp accessibility enable
cdp accessibility disable
cdp accessibility getFullAXTree
cdp accessibility getPartialAXTree --nodeId <id>
cdp accessibility queryAXTree --accessibleName <name>
```

#### Security

```bash
cdp security enable
cdp security disable
cdp security setIgnoreCertificateErrors --ignore
```

#### ServiceWorker

```bash
cdp serviceworker enable
cdp serviceworker disable
cdp serviceworker inspectWorker --versionId <id>
cdp serviceworker skipWaiting --scopeURL <url>
cdp serviceworker unregister --scopeURL <url>
```

#### LayerTree

```bash
cdp layertree enable
cdp layertree disable
cdp layertree compositingReasons --layerId <id>
cdp layertree makeSnapshot --layerId <id>
```

#### Animation

```bash
cdp animation enable
cdp animation disable
cdp animation getCurrentTime --id <id>
cdp animation setPaused --animations <ids> --paused
cdp animation setPlaybackRate --playbackRate <n>
```

#### Media

```bash
cdp media enable
cdp media disable
```

#### WebAudio

```bash
cdp webaudio enable
cdp webaudio disable
cdp webaudio getRealtimeData --contextId <id>
```

#### Storage

```bash
cdp storage clearDataForOrigin --origin <origin> --storageTypes <types>
cdp storage getCookies --browserContextId <id>
cdp storage getUsageAndQuota --origin <origin>
```

#### Console

```bash
cdp console enable
cdp console disable
cdp console clearMessages
```

#### Log

```bash
cdp log enable
cdp log disable
cdp log clear
```

#### Performance

```bash
cdp performance enable
cdp performance disable
cdp performance getMetrics
```

#### Profiler

```bash
cdp profiler enable
cdp profiler disable
cdp profiler start
cdp profiler stop
cdp profiler getBestEffortCoverage
cdp profiler startPreciseCoverage [--callCount] [--detailed]
cdp profiler stopPreciseCoverage
cdp profiler takePreciseCoverage
```

#### HeapProfiler

```bash
cdp heapprofiler enable
cdp heapprofiler disable
cdp heapprofiler takeHeapSnapshot
cdp heapprofiler collectGarbage
cdp heapprofiler getHeapObjectId --objectId <id>
```

### Event Subscriptions

For any domain, subscribe to events:

```bash
cdp events subscribe --domain Network [--events requestWillBeSent,responseReceived]
cdp events unsubscribe --domain Network
cdp events list  # List active subscriptions
```

Events stream as NDJSON to stdout.

---

## High-Level Commands

Composed operations that reduce agent round-trips and context pollution.

### Resource Sync

Sync page resources to local SQLite database for efficient querying.

```bash
# Full sync of current page
cdp sync resources --db ./site.db

# Sync specific resource types
cdp sync resources --db ./site.db --types html,css,js,images

# Sync with source maps resolved
cdp sync resources --db ./site.db --resolve-sourcemaps

# Watch mode - continuously sync as resources change
cdp sync resources --db ./site.db --watch --output ndjson
```

### DOM Operations

```bash
# Get full DOM as structured JSON
cdp snapshot dom [--styles] [--computed] [--boxmodel]

# Get accessibility tree (often more useful for agents than raw DOM)
cdp snapshot a11y [--interesting-only] [--root "<selector>"]

# Get page as text (extracted readable content)
cdp snapshot text [--selector "<sel>"]

# Query and return matching elements with context
cdp query "<selector>" [--limit <n>] [--include-styles] [--include-children <depth>]

# Diff DOM state between two snapshots
cdp diff dom --before <snapshot1.json> --after <snapshot2.json>

# Batch DOM operations
cdp dom batch --operations <json-file>
```

### Style Analysis

```bash
# Extract all styles affecting an element
cdp styles for "<selector>" [--computed] [--inherited] [--pseudo]

# Get all stylesheets as structured data
cdp styles sheets [--inline] [--external]

# Find unused CSS rules
cdp styles unused [--threshold <percent>]

# Get CSS coverage data
cdp styles coverage
```

### Network Operations

```bash
# Capture all network activity for a navigation
cdp capture navigation --url <url> --output <har-file>

# Watch requests matching pattern (streams NDJSON)
cdp watch requests [--url-pattern <pattern>] [--method <method>] [--type xhr|fetch|script|stylesheet]

# Get all resources loaded by page as manifest
cdp manifest resources [--include-data] [--hash]

# Block URLs matching pattern
cdp network block --patterns "*.analytics.com/*,*.ads.*"

# Unblock
cdp network unblock

# Throttle network
cdp network throttle slow-3g|fast-3g|offline
cdp network throttle --latency 100 --download 1000000

# Set extra headers for all requests
cdp network headers --set "Authorization: Bearer token123"
cdp network headers --clear
```

### Request Interception

Use the Fetch domain for request/response interception (Network.setRequestInterception is deprecated).

```bash
# Start interception with URL patterns
cdp intercept start --patterns "*/api/*,*.json"

# Start interception at response stage (to modify response)
cdp intercept start --patterns "*/api/*" --stage response

# Process intercepted requests (streams NDJSON, expects commands on stdin)
cdp intercept handle
# Input:  {"action": "continue", "requestId": "123"}
# Input:  {"action": "fulfill", "requestId": "456", "status": 200, "body": "{...}"}
# Input:  {"action": "fail", "requestId": "789", "reason": "BlockedByClient"}

# Apply interception rules from file
cdp intercept --rules rules.json

# Stop interception
cdp intercept stop
```

Rules file format:

```json
{
  "rules": [
    {
      "match": {"url": "*/api/user*"},
      "action": "fulfill",
      "response": {"status": 200, "body": {"mocked": true}}
    },
    {
      "match": {"url": "*.analytics.*"},
      "action": "fail",
      "reason": "BlockedByClient"
    },
    {
      "match": {"url": "*", "method": "POST"},
      "action": "modify",
      "headers": {"X-Intercepted": "true"}
    },
    {
      "match": {"url": "*/api/data*"},
      "action": "delay",
      "delay_ms": 2000
    }
  ]
}
```

### Script Operations

```bash
# Evaluate JS and return structured result
cdp eval "<expression>" [--await] [--serialize-depth <n>]

# Evaluate JS file
cdp eval --file <script.js>

# Get all scripts with source
cdp scripts list [--include-source] [--include-sourcemaps]

# Get code coverage for scripts
cdp scripts coverage [--format lcov|json]
```

### Page Operations

```bash
# Navigate and wait for load
cdp goto <url> [--wait-until load|domcontentloaded|networkidle]

# Take screenshot with options
cdp screenshot [--selector "<sel>"] [--full-page] [--format png|jpeg|webp]

# Generate PDF
cdp pdf [--format A4|Letter] [--landscape] [--print-background]

# Get page metrics (performance, resources, etc.)
cdp metrics [--categories performance,resources,coverage]

# List all open pages/tabs
cdp tabs [--output json]

# Select active page by title/url pattern
cdp select "<pattern>"
```

### Input Automation (High-Level)

Simplified interaction commands that handle waiting and coordinate mapping.

```bash
# Click element (finds element, scrolls into view, clicks center)
cdp click "<selector>" [--user-gesture] [--button left|right|middle]

# Fill single input
cdp fill "<selector>" "<text>" [--clear]

# Fill entire form from JSON
cdp fill-form --data '{"#email": "test@example.com", "#password": "secret"}'
cdp fill-form --file form-data.json

# Type text with realistic timing
cdp type "<text>" [--delay <ms>]

# Press key or key combination
cdp key "<key>" [--modifiers ctrl,shift,alt,meta]
# Examples: cdp key Enter, cdp key "ctrl+s", cdp key Escape

# Hover over element
cdp hover "<selector>"

# Drag element to target
cdp drag "<from-selector>" "<to-selector>"
cdp drag "<from-selector>" --x <x> --y <y>

# File upload
cdp upload "<selector>" <file-path> [<file-path>...]

# Handle dialogs (alert, confirm, prompt)
cdp dialog accept [--text "<prompt-response>"]
cdp dialog dismiss
cdp dialog --auto-accept   # Auto-accept all dialogs
cdp dialog --auto-dismiss  # Auto-dismiss all dialogs

# Scroll
cdp scroll --to "<selector>"
cdp scroll --by <x> <y>
cdp scroll --to-top
cdp scroll --to-bottom
```

### Wait Operations

```bash
# Wait for element
cdp wait "<selector>" [--visible] [--hidden] [--timeout <ms>]

# Wait for navigation
cdp wait navigation [--url "<pattern>"]

# Wait for network idle
cdp wait idle [--timeout <ms>]

# Wait for specific network request
cdp wait request --url "<pattern>" [--method <method>]

# Wait for console message
cdp wait console --text "<pattern>" [--level error|warn|log]

# Wait for arbitrary condition
cdp wait eval "<js-expression>" [--poll <ms>] [--timeout <ms>]
```

### Performance Tracing

```bash
# Start trace recording
cdp trace start [--categories loading,rendering,scripting]

# Stop and save trace
cdp trace stop --output trace.json

# Analyze trace for insights
cdp trace analyze trace.json [--focus lcp|fcp|cls|tbt]
```

### Core Web Vitals

```bash
# Get live Core Web Vitals metrics
cdp vitals [--watch]

# Output:
# {"lcp": {"value": 1234, "rating": "good"}, "cls": {"value": 0.05, "rating": "good"}, "fcp": {...}}

# Measure vitals for a navigation
cdp vitals measure --url <url> [--interactions <count>]

# Get TTFB for current page
cdp vitals ttfb
```

### HAR Export

```bash
# Capture network activity as HAR file
cdp har capture --output traffic.har [--sanitize]

# Start HAR recording (async)
cdp har start

# Stop and save
cdp har stop --output traffic.har

# Convert existing network events to HAR
cdp har export --from-db ./site.db --output traffic.har
```

### Coverage Analysis

```bash
# Start coverage collection
cdp coverage start [--detailed] [--call-count]

# Stop and get report
cdp coverage stop

# Get CSS coverage
cdp coverage css [--threshold 50]  # Warn if >50% unused

# Get JS coverage
cdp coverage js [--format json|lcov]

# Find unused CSS rules
cdp coverage unused-css [--output unused.json]

# Interactive: reload and measure
cdp coverage measure --url <url> [--interact]
```

### Storage Management

```bash
# Cookies
cdp storage cookies [--domain <domain>]           # List cookies
cdp storage cookies set --name <n> --value <v> --domain <d> [--httpOnly] [--secure]
cdp storage cookies delete --name <n> --domain <d>
cdp storage cookies clear [--domain <domain>]
cdp storage cookies export --output cookies.json  # Export all cookies
cdp storage cookies import --file cookies.json    # Import cookies

# LocalStorage
cdp storage local [--origin <origin>]             # List localStorage items
cdp storage local get <key> [--origin <origin>]
cdp storage local set <key> <value> [--origin <origin>]
cdp storage local delete <key> [--origin <origin>]
cdp storage local clear [--origin <origin>]

# SessionStorage
cdp storage session [--origin <origin>]           # List sessionStorage items
cdp storage session get <key>
cdp storage session set <key> <value>
cdp storage session clear

# IndexedDB
cdp storage indexeddb list [--origin <origin>]    # List databases
cdp storage indexeddb dump <db> [--store <store>] # Dump contents
cdp storage indexeddb clear <db> [--origin <origin>]

# Cache Storage (Service Worker caches)
cdp storage cache list                            # List caches
cdp storage cache contents <cache-name>           # List cached resources
cdp storage cache delete <cache-name>

# Clear all storage for origin
cdp storage clear --origin <origin> [--types cookies,local,session,indexeddb,cache]
```

### Memory Analysis

```bash
# Take heap snapshot
cdp memory snapshot --output heap.json

# Compare two snapshots (find leaks)
cdp memory diff --before heap1.json --after heap2.json

# Get memory usage summary
cdp memory usage

# Start allocation tracking
cdp memory track start

# Stop and get allocation report
cdp memory track stop [--output allocations.json]

# Force garbage collection
cdp memory gc

# Prepare for leak detection (stops workers, clears caches)
cdp memory prepare-leak-check
```

### Device Emulation

```bash
# Emulate preset device
cdp emulate device "iPhone 14"
cdp emulate device "Pixel 7"
cdp emulate device "iPad Pro"

# List available device presets
cdp emulate devices --list

# Custom viewport
cdp emulate viewport --width 375 --height 812 --dpr 3 --mobile

# Emulate network conditions
cdp emulate network slow-3g
cdp emulate network fast-3g
cdp emulate network offline
cdp emulate network --latency 100 --download 1000000 --upload 500000

# Emulate CPU throttling
cdp emulate cpu --slowdown 4  # 4x slower

# Emulate geolocation
cdp emulate geo --lat 37.7749 --lng -122.4194

# Emulate timezone
cdp emulate timezone "America/New_York"

# Emulate vision deficiency
cdp emulate vision blurred|protanopia|deuteranopia|tritanopia|achromatopsia

# Clear all emulation
cdp emulate reset
```

### Service Worker / PWA

```bash
# List registered service workers
cdp sw list

# Get service worker info
cdp sw info <registration-id>

# Force update service worker
cdp sw update <scope-url>

# Skip waiting (activate waiting worker)
cdp sw skip-waiting <scope-url>

# Unregister service worker
cdp sw unregister <scope-url>

# Inspect service worker (attach debugger)
cdp sw inspect <version-id>

# Trigger background sync
cdp sw sync <tag>

# Trigger push message
cdp sw push [--data <json>]
```

### Accessibility

```bash
# Get full accessibility tree
cdp a11y tree [--root "<selector>"] [--interesting-only]

# Query by accessible name/role
cdp a11y query --name "Submit"
cdp a11y query --role button

# Check element accessibility
cdp a11y check "<selector>"

# Emulate vision deficiencies (for testing)
cdp a11y emulate protanopia|deuteranopia|tritanopia|achromatopsia

# Run accessibility audit (axe-core integration)
cdp a11y audit [--rules wcag2a,wcag2aa] [--include "<selector>"] [--exclude "<selector>"]
```

### Source Maps

```bash
# List scripts with source map info
cdp sourcemap list

# Resolve source map for a script
cdp sourcemap resolve <script-id>

# Map position from generated to original
cdp sourcemap position <script-id> --line <n> --column <n>

# Download and cache all source maps
cdp sourcemap fetch-all --output ./sourcemaps/

# Get original source file
cdp sourcemap original <script-id> --file <original-path>
```

### Selector Tools

Generate and validate selectors for automation.

```bash
# Generate selector for element at screen coordinates
cdp selector at --x 100 --y 200 [--types css,xpath,playwright]

# Generate selectors for a queried element
cdp selector for "<approximate-selector>" [--unique] [--types all]

# Output:
# {"css": "button.submit", "xpath": "//button[@class='submit']", "playwright": "button:has-text('Submit')"}

# Find unique/optimal selector for node
cdp selector unique --nodeId <id>

# Validate selector (returns match count)
cdp selector test "<selector>" [--expected <n>]

# Generate selector from accessibility properties
cdp selector a11y --role button --name "Submit"

# Copy selector to clipboard (if available)
cdp selector copy "<selector-type>" --nodeId <id>
```

### Form Profiles

Save and replay form states for testing.

```bash
# Generate random test data for form
cdp form fill-random [--locale en-US] [--seed <n>]

# Save current form state as profile
cdp form save "<form-selector>" --name "checkout-test" --output profiles/

# Load and apply form profile
cdp form load --profile profiles/checkout-test.json

# List saved profiles
cdp form profiles [--path ./profiles]

# Record form interactions (watch mode)
cdp form record --output recorded.json
# Stops on Ctrl+C, captures all form changes

# Replay recorded interactions
cdp form replay --file recorded.json [--delay <ms>]
```

### PDF Export

```bash
# Generate PDF from current page
cdp pdf --output page.pdf

# PDF options
cdp pdf --output page.pdf \
  --format A4|Letter|Legal \
  --landscape \
  --print-background \
  --scale 0.8 \
  --margin-top 1cm \
  --header-template "<div>Header</div>" \
  --footer-template "<div>Page <span class=pageNumber></span></div>"

# PDF from specific element
cdp pdf --selector "#content" --output content.pdf
```

### Clipboard

```bash
# Read clipboard content
cdp clipboard read

# Write to clipboard
cdp clipboard write "text content"

# Write file contents to clipboard
cdp clipboard write --file ./data.json
```

### Session Management

```bash
# Create named session with specific configuration
cdp session create <name> --port 9222 --default-timeout 30s

# List sessions
cdp session list

# Use session
cdp --session <name> <command>

# Destroy session
cdp session destroy <name>
```

### Batch Operations

Execute multiple operations in a single command to reduce round-trips:

```bash
# Execute batch from file
cdp batch --file operations.json

# Execute batch from stdin
echo '[{"cmd":"page.navigate","args":{"url":"..."}},...]' | cdp batch --stdin
```

Batch file format:

```json
{
  "operations": [
    { "id": "1", "cmd": "page.navigate", "args": { "url": "https://example.com" } },
    { "id": "2", "cmd": "dom.getDocument", "args": { "depth": -1 }, "depends": ["1"] },
    { "id": "3", "cmd": "css.getAllStyleSheets", "depends": ["1"] }
  ],
  "parallel": true
}
```

---

## SQLite Schema

For local resource storage and querying.

### Core Tables

```sql
-- Sync metadata
CREATE TABLE sync_meta (
    id INTEGER PRIMARY KEY,
    page_url TEXT NOT NULL,
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    chrome_version TEXT,
    user_agent TEXT
);

-- Resources (scripts, stylesheets, images, etc.)
CREATE TABLE resources (
    id INTEGER PRIMARY KEY,
    sync_id INTEGER REFERENCES sync_meta(id),
    url TEXT NOT NULL,
    type TEXT NOT NULL,  -- 'script', 'stylesheet', 'image', 'font', 'document', 'xhr', 'fetch', 'websocket', 'wasm', 'other'
    mime_type TEXT,
    status_code INTEGER,
    headers TEXT,  -- JSON
    content BLOB,
    content_hash TEXT,  -- SHA256
    size INTEGER,
    from_cache BOOLEAN,
    timing TEXT,  -- JSON (request timing data)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(sync_id, url)
);

CREATE INDEX idx_resources_type ON resources(type);
CREATE INDEX idx_resources_url ON resources(url);
CREATE INDEX idx_resources_hash ON resources(content_hash);

-- Scripts (parsed/analyzed JS)
CREATE TABLE scripts (
    id INTEGER PRIMARY KEY,
    resource_id INTEGER REFERENCES resources(id),
    script_id TEXT,  -- CDP script ID
    source TEXT,
    source_map_url TEXT,
    source_map TEXT,  -- Resolved source map content
    is_module BOOLEAN,
    execution_context_id INTEGER,
    start_line INTEGER,
    start_column INTEGER,
    end_line INTEGER,
    end_column INTEGER,
    hash TEXT
);

CREATE INDEX idx_scripts_resource ON scripts(resource_id);

-- Stylesheets (parsed CSS)
CREATE TABLE stylesheets (
    id INTEGER PRIMARY KEY,
    resource_id INTEGER REFERENCES resources(id),
    stylesheet_id TEXT,  -- CDP stylesheet ID
    source TEXT,
    source_map_url TEXT,
    frame_id TEXT,
    origin TEXT,  -- 'injected', 'user-agent', 'inspector', 'regular'
    is_inline BOOLEAN,
    start_line INTEGER,
    start_column INTEGER,
    owner_node_id INTEGER
);

CREATE INDEX idx_stylesheets_resource ON stylesheets(resource_id);

-- DOM Snapshots
CREATE TABLE dom_snapshots (
    id INTEGER PRIMARY KEY,
    sync_id INTEGER REFERENCES sync_meta(id),
    root_node TEXT,  -- JSON (full DOM tree)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Individual DOM nodes (flattened for querying)
CREATE TABLE dom_nodes (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES dom_snapshots(id),
    node_id INTEGER,  -- CDP node ID
    parent_node_id INTEGER,
    node_type INTEGER,
    node_name TEXT,
    node_value TEXT,
    attributes TEXT,  -- JSON
    selector_path TEXT,  -- Unique CSS selector path
    xpath TEXT,
    outer_html TEXT,
    bounding_box TEXT  -- JSON {x, y, width, height}
);

CREATE INDEX idx_dom_nodes_snapshot ON dom_nodes(snapshot_id);
CREATE INDEX idx_dom_nodes_name ON dom_nodes(node_name);
CREATE INDEX idx_dom_nodes_selector ON dom_nodes(selector_path);

-- CSS Rules (flattened for querying)
CREATE TABLE css_rules (
    id INTEGER PRIMARY KEY,
    stylesheet_id INTEGER REFERENCES stylesheets(id),
    selector TEXT,
    specificity TEXT,  -- JSON [a, b, c]
    properties TEXT,  -- JSON
    source_line INTEGER,
    source_column INTEGER,
    is_media_rule BOOLEAN,
    media_query TEXT
);

CREATE INDEX idx_css_rules_stylesheet ON css_rules(stylesheet_id);
CREATE INDEX idx_css_rules_selector ON css_rules(selector);

-- Console messages
CREATE TABLE console_messages (
    id INTEGER PRIMARY KEY,
    sync_id INTEGER REFERENCES sync_meta(id),
    level TEXT,  -- 'log', 'warning', 'error', 'debug', 'info'
    text TEXT,
    url TEXT,
    line INTEGER,
    column INTEGER,
    timestamp DATETIME,
    stack_trace TEXT  -- JSON
);

CREATE INDEX idx_console_level ON console_messages(level);

-- Network requests (detailed HAR-like data)
CREATE TABLE network_requests (
    id INTEGER PRIMARY KEY,
    sync_id INTEGER REFERENCES sync_meta(id),
    request_id TEXT,
    url TEXT,
    method TEXT,
    request_headers TEXT,  -- JSON
    request_body TEXT,
    response_status INTEGER,
    response_headers TEXT,  -- JSON
    response_body_resource_id INTEGER REFERENCES resources(id),
    timing TEXT,  -- JSON
    initiator TEXT,  -- JSON
    type TEXT,
    from_cache BOOLEAN,
    from_service_worker BOOLEAN,
    timestamp DATETIME
);

CREATE INDEX idx_network_url ON network_requests(url);
CREATE INDEX idx_network_status ON network_requests(response_status);
```

### Coverage Tables

```sql
-- Coverage snapshots
CREATE TABLE coverage_snapshots (
    id INTEGER PRIMARY KEY,
    sync_id INTEGER REFERENCES sync_meta(id),
    type TEXT NOT NULL,  -- 'css', 'js'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Coverage ranges
CREATE TABLE coverage_ranges (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES coverage_snapshots(id),
    resource_id INTEGER REFERENCES resources(id),
    start_offset INTEGER,
    end_offset INTEGER,
    count INTEGER,  -- For JS: execution count. For CSS: 1 if used, 0 if unused
    function_name TEXT  -- For JS coverage
);

CREATE INDEX idx_coverage_snapshot ON coverage_ranges(snapshot_id);
CREATE INDEX idx_coverage_resource ON coverage_ranges(resource_id);
```

### Performance Tables

```sql
-- Performance metrics snapshots
CREATE TABLE performance_snapshots (
    id INTEGER PRIMARY KEY,
    sync_id INTEGER REFERENCES sync_meta(id),
    url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Core Web Vitals and metrics
CREATE TABLE performance_metrics (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES performance_snapshots(id),
    metric_name TEXT NOT NULL,  -- 'LCP', 'FCP', 'CLS', 'TTFB', 'INP', 'TBT'
    value REAL,
    rating TEXT,  -- 'good', 'needs-improvement', 'poor'
    element_selector TEXT,  -- For LCP: the element
    attribution TEXT  -- JSON with additional context
);

CREATE INDEX idx_perf_snapshot ON performance_metrics(snapshot_id);
CREATE INDEX idx_perf_metric ON performance_metrics(metric_name);

-- Trace events (for detailed analysis)
CREATE TABLE trace_events (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES performance_snapshots(id),
    name TEXT,
    category TEXT,
    phase TEXT,
    timestamp INTEGER,
    duration INTEGER,
    args TEXT  -- JSON
);

CREATE INDEX idx_trace_snapshot ON trace_events(snapshot_id);
CREATE INDEX idx_trace_name ON trace_events(name);
```

### Storage Snapshots

```sql
-- Storage snapshots for restoration
CREATE TABLE storage_snapshots (
    id INTEGER PRIMARY KEY,
    origin TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Cookies snapshot
CREATE TABLE cookies_snapshot (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES storage_snapshots(id),
    name TEXT,
    value TEXT,
    domain TEXT,
    path TEXT,
    expires DATETIME,
    http_only BOOLEAN,
    secure BOOLEAN,
    same_site TEXT
);

-- LocalStorage snapshot
CREATE TABLE localstorage_snapshot (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES storage_snapshots(id),
    key TEXT,
    value TEXT
);

-- SessionStorage snapshot
CREATE TABLE sessionstorage_snapshot (
    id INTEGER PRIMARY KEY,
    snapshot_id INTEGER REFERENCES storage_snapshots(id),
    key TEXT,
    value TEXT
);
```

### Future: Vector Search Tables

```sql
-- Embeddings for semantic search (requires sqlite-vec)
CREATE VIRTUAL TABLE embeddings USING vec0(
    id INTEGER PRIMARY KEY,
    content_type TEXT,  -- 'script', 'style', 'dom_node', 'comment'
    content_id INTEGER,  -- FK to relevant table
    embedding FLOAT[384]  -- Dimension depends on model
);

-- Code symbols (for code intelligence)
CREATE TABLE code_symbols (
    id INTEGER PRIMARY KEY,
    script_id INTEGER REFERENCES scripts(id),
    name TEXT,
    kind TEXT,  -- 'function', 'class', 'variable', 'import', 'export'
    start_line INTEGER,
    start_column INTEGER,
    end_line INTEGER,
    end_column INTEGER,
    parent_symbol_id INTEGER REFERENCES code_symbols(id),
    signature TEXT,
    doc_comment TEXT
);

CREATE INDEX idx_symbols_script ON code_symbols(script_id);
CREATE INDEX idx_symbols_name ON code_symbols(name);
CREATE INDEX idx_symbols_kind ON code_symbols(kind);
```

---

## CLI Global Flags

```bash
cdp [global-flags] <command> [command-flags]

Global Flags:
  --port <n>           Chrome debug port (default: 9222, env: CDP_PORT)
  --host <host>        Chrome debug host (default: localhost, env: CDP_HOST)
  --ws <url>           WebSocket URL directly (overrides host/port)
  --target <id|url>    Target specific page by ID or URL pattern
  --session <name>     Use named session
  --output <format>    Output format: json, ndjson, text (default: json)
  --timeout <duration> Command timeout (default: 30s)
  --quiet              Suppress non-essential output
  --verbose            Enable verbose logging
  --config <file>      Config file path (default: ~/.cdp/config.yaml)
  --list               List available subcommands/domains
  --describe           Show detailed help for command
  --search <query>     Search commands/methods
```

### Command Matching

Commands are **case-insensitive** for ease of use:

```bash
cdp DOM querySelector        # Works
cdp dom queryselector        # Also works
cdp Network getResponseBody  # Works
cdp network getresponsebody  # Also works
```

---

## Configuration File

`~/.cdp/config.yaml`:

```yaml
default:
  port: 9222
  host: localhost
  timeout: 30s
  output: json

sessions:
  dev:
    port: 9222
    timeout: 60s

  staging:
    host: staging.local
    port: 9223

aliases:
  nav: "page navigate"
  doc: "dom getDocument --depth -1"
  ss: "screenshot --full-page"

# SQLite defaults
database:
  path: ~/.cdp/default.db

# Event subscriptions to auto-enable
auto_subscribe:
  - domain: Console
  - domain: Log
```

---

## Agent Integration Patterns

### Skill Context Management

The CLI is designed to minimize context pollution for agent skills:

1. **Query locally first** - Sync to SQLite, then query without CDP round-trips
2. **Batch operations** - Combine multiple operations in single call
3. **Structured output** - JSON responses can be filtered/summarized before presenting to agent
4. **Event streaming** - NDJSON events can be filtered in real-time

### Example Skill Workflow

```bash
# 1. Initial page load and sync
cdp goto "https://example.com" --wait-until networkidle
cdp sync resources --db ./page.db

# 2. Agent queries local database (no CDP calls)
sqlite3 ./page.db "SELECT url, size FROM resources WHERE type='script'"

# 3. Agent needs specific element
cdp query "button.submit" --include-styles

# 4. Agent modifies DOM
cdp dom setOuterHTML --nodeId 123 --outerHTML '<button class="submit">Updated</button>'

# 5. Agent captures result
cdp screenshot --selector "button.submit" --format png
```

### Error Recovery

All commands are designed for safe retry:

```bash
# Idempotent - safe to retry
cdp page navigate --url "https://example.com"
cdp network enable

# Returns current state if already done
cdp dom getDocument
```

### Agent-Friendly Patterns

**1. Progressive disclosure** - Start with minimal output, add detail as needed:
```bash
# Agent starts with overview
cdp tabs

# Selects target page
cdp select "GitHub"

# Gets accessibility tree (often sufficient for understanding page)
cdp snapshot a11y

# Only fetches full DOM if needed
cdp snapshot dom --depth 3
```

**2. Local-first querying** - Sync once, query many times:
```bash
# Sync page state to SQLite
cdp sync resources --db ./state.db

# Agent can now query without CDP round-trips
sqlite3 ./state.db "SELECT selector_path FROM dom_nodes WHERE node_name='BUTTON'"
sqlite3 ./state.db "SELECT url FROM resources WHERE type='script' AND size > 100000"
```

**3. Batch operations** - Reduce context and round-trips:
```bash
# Instead of multiple commands
cdp fill "#email" "test@test.com"
cdp fill "#password" "secret"
cdp click "[type=submit]"

# Single atomic operation
cdp fill-form --data '{"#email":"test@test.com","#password":"secret"}' && cdp click "[type=submit]"
```

**4. User gesture mode** - Handle activation-gated APIs:
```bash
# Some APIs require user gesture (WebXR, fullscreen, clipboard)
cdp click "#start-vr" --user-gesture
```

---

## Implementation Phases

### Phase 1: Foundation
- [ ] CLI scaffolding (cobra)
- [ ] CDP connection management (chromedp)
- [ ] Chrome launch helper with channel selection
- [ ] Output formatting (JSON/NDJSON/text)
- [ ] Error handling with semantic exit codes
- [ ] Configuration system (viper)
- [ ] Self-discovery (--list, --describe, --search)

### Phase 2: Low-Level Commands
- [ ] Code generation from CDP protocol JSON
- [ ] All 65+ CDP domains exposed as commands
- [ ] Case-insensitive command matching
- [ ] Event subscription system (NDJSON streaming)
- [ ] Batch operation support

### Phase 3: High-Level Navigation & Input
- [ ] `tabs`, `select`, `goto` commands
- [ ] `click`, `fill`, `fill-form`, `type`, `key`
- [ ] `hover`, `drag`, `scroll`
- [ ] `upload` for file inputs
- [ ] `dialog` handling (accept/dismiss/auto)
- [ ] `wait` operations (element, navigation, idle, eval)

### Phase 4: Snapshots & Inspection
- [ ] `snapshot dom` with styles
- [ ] `snapshot a11y` (accessibility tree)
- [ ] `snapshot text` (readable content)
- [ ] `screenshot` with element selection, full-page
- [ ] `query` with context
- [ ] Console with progressive verbosity
- [ ] Object inspection (--inspect, --depth)

### Phase 5: Network & Interception
- [ ] `watch requests` with filtering
- [ ] `har capture/start/stop/export`
- [ ] `intercept start/handle/stop`
- [ ] Request interception rules engine
- [ ] `network block/throttle/headers`

### Phase 6: Storage Management
- [ ] `storage cookies` CRUD operations
- [ ] `storage local/session` management
- [ ] `storage indexeddb` listing and dump
- [ ] `storage cache` (Service Worker caches)
- [ ] Storage snapshot/restore for session persistence

### Phase 7: SQLite Integration
- [ ] Schema implementation (resources, DOM, network, coverage, performance)
- [ ] `sync resources` command
- [ ] DOM snapshot storage
- [ ] Network request storage (HAR-like)
- [ ] Console message storage
- [ ] Query helpers

### Phase 8: Performance & Coverage
- [ ] `vitals` - Core Web Vitals (LCP, CLS, FCP, TTFB)
- [ ] `trace start/stop/analyze`
- [ ] `coverage css/js` with unused detection
- [ ] Source map resolution and mapping
- [ ] `memory snapshot/diff/gc`

### Phase 9: Device Emulation & Responsive
- [ ] Device presets (iPhone, Pixel, iPad, etc.)
- [ ] Custom viewport with DPR and touch
- [ ] Network condition emulation
- [ ] CPU throttling
- [ ] Geolocation and timezone
- [ ] Vision deficiency emulation
- [ ] `responsive capture/breakpoints/diff`

### Phase 10: Accessibility & PWA
- [ ] `a11y tree/query/check/audit`
- [ ] axe-core integration for WCAG audits
- [ ] `sw list/update/skip-waiting/unregister`
- [ ] Background sync/push triggers

### Phase 11: Advanced Features
- [ ] Selector generation and validation
- [ ] Form recording and profile management
- [ ] Random test data generation (faker-like)
- [ ] JSON response formatting and viewing
- [ ] PDF generation with options

### Phase 12: Future (Vector Search & AI)
- [ ] sqlite-vec integration
- [ ] Embedding generation for content
- [ ] Semantic search commands
- [ ] Code symbol extraction
- [ ] AST-based JavaScript/CSS analysis

---

## Dependencies

```go
require (
    github.com/spf13/cobra v1.8+
    github.com/spf13/viper v1.18+
    github.com/chromedp/chromedp v0.9+
    github.com/chromedp/cdproto v0.0.0+
    modernc.org/sqlite v1.28+
)
```
