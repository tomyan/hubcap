# hubcap inspect — Terminal Web Inspector

A terminal-based Chrome DevTools inspector, launched as `hubcap inspect`. Connects to a Chrome target via CDP and presents an interactive TUI built with sumi + sumi-ui.

## Overview

```
hubcap inspect [--port 9222] [--target <id>]
```

Opens a full-screen terminal UI with tabbed panels mirroring Chrome DevTools. Uses hubcap's existing CDP client for all browser communication. The UI is built with sumi (framework) and sumi-ui (component library).

## Architecture

```
hubcap inspect command
  ├── CDP client (existing hubcap internal/chrome)
  ├── sumi TUI (framework)
  ├── sumi-ui components (TreeView, Table, TabPanel, etc.)
  └── panel modules (Console, Elements, Network, etc.)
```

Each panel is a sumi component that:
1. Subscribes to CDP events via the existing client
2. Stores data in signals
3. Renders via sumi-ui components

The top-level inspector component manages:
- Tab bar for panel switching
- Global keyboard shortcuts
- CDP connection lifecycle
- StatusBar with connection info and shortcut hints

## Panels — Priority Order

### Panel 1: Console

The simplest panel and the one that provides immediate value.

**Features:**
- Streaming console output (log, warn, error, info, debug)
- Severity-level colouring and icons
- REPL input at the bottom — evaluate JS expressions
- Filter by text and severity level
- Object inspection (expandable tree for objects/arrays)
- Error stack traces

**CDP domains:** `Runtime` (enable, evaluate, consoleAPICalled, exceptionThrown), `Log` (enable, entryAdded)

**sumi-ui components:** ScrollableLog, FilterBar, TextInput, TreeView (for object inspection)

**Layout:**
```
┌─ Console ────────────────────────────────┐
│ [Filter: ___________] [V I W E]          │  <- FilterBar with severity toggles
│──────────────────────────────────────────│
│ > page loaded                            │  <- ScrollableLog
│ ⚠ deprecated API call at foo.js:12       │
│ ✕ TypeError: cannot read property 'x'    │
│   at bar (foo.js:45)                     │
│   at baz (app.js:12)                     │
│ ▸ {name: "test", items: Array(3)}        │  <- expandable object
│                                          │
│──────────────────────────────────────────│
│ > _                                      │  <- REPL input
└──────────────────────────────────────────┘
```

### Panel 2: Elements

The most complex but highest-value panel. Shows the live DOM tree and CSS styles.

**Features:**
- Collapsible DOM tree with syntax-coloured tags/attributes
- Live updates as DOM mutates
- Styles pane — matched CSS rules with specificity ordering
- Computed pane — resolved values, filterable
- Box model diagram (margin/border/padding/content)
- Layout pane — flex/grid properties
- Event listeners list

**CDP domains:** `DOM` (getDocument, requestChildNodes, setAttributeValue, setNodeValue), `CSS` (getMatchedStylesForNode, getComputedStyleForNode), `DOMDebugger` (getEventListeners)

**sumi-ui components:** TreeView, PropertyList, BoxModel, TabPanel, FilterBar

**Layout:**
```
┌─ Elements ───────────────────────────────────────────────────┐
│ ▾ <html lang="en">                │ Styles | Computed | Layout│
│   ▾ <head>                        │────────────────────────── │
│     <meta charset="utf-8">        │ element.style {           │
│     <title>My Page</title>        │ }                         │
│   ▸ <body class="dark">           │ .container {              │
│                                   │   display: flex;          │
│                                   │   padding: 16px;          │
│                                   │ }                         │
│                                   │──── Box Model ─────────── │
│                                   │  ┌─ margin: 0 ─────────┐ │
│                                   │  │ ┌─ border: 1 ──────┐ │ │
│                                   │  │ │ ┌─ padding: 16 ─┐ │ │ │
│                                   │  │ │ │  200 × 100    │ │ │ │
└──────────────────────────────────────────────────────────────┘
```

### Panel 3: Network

Real-time network request monitoring with detail inspection.

**Features:**
- Request list table (Name, Status, Type, Size, Time)
- Filter by type (XHR, JS, CSS, Img, Font, Doc, WS) and text
- Request detail tabs: Headers, Payload, Preview, Response, Timing
- Timing waterfall bars in the table
- Preserve log toggle
- Clear button

**CDP domains:** `Network` (enable, requestWillBeSent, responseReceived, loadingFinished, getResponseBody)

**sumi-ui components:** Table, TabPanel, FilterBar, PropertyList, CodeViewer (for response preview)

**Layout:**
```
┌─ Network ────────────────────────────────────────────────────┐
│ [Filter: ___________] [XHR JS CSS Img Font Doc WS All]       │
│──────────────────────────────────────────────────────────────│
│ Name              Status  Type  Size   Time   Waterfall      │
│ index.html          200   doc   4.2K   120ms  ██             │
│ styles.css          200   css   1.1K    45ms    ██           │
│▸api/users           200   xhr   890B   230ms      ████       │
│ app.js              200   js   12.4K    89ms   ███           │
│──────────────────────────────────────────────────────────────│
│ Headers | Payload | Preview | Response | Timing              │
│ Request URL: https://example.com/api/users                   │
│ Request Method: GET                                          │
│ Status Code: 200 OK                                          │
│ Content-Type: application/json                               │
└──────────────────────────────────────────────────────────────┘
```

### Panel 4: Sources

Source file browser with read-only viewer and breakpoint support.

**Features:**
- File tree (by origin/domain)
- Source viewer with line numbers and syntax highlighting
- Breakpoint gutter markers
- Watch expressions
- Call stack (when paused)
- Scope inspection (when paused)

**CDP domains:** `Debugger` (enable, setBreakpointByUrl, pause, resume, stepOver, stepInto, stepOut, evaluateOnCallFrame, getScriptSource), `Runtime` (getProperties), `Page` (getResourceTree)

**sumi-ui components:** TreeView, CodeViewer, PropertyList, TabPanel

### Panel 5: Application

Storage browser for cookies, localStorage, sessionStorage, IndexedDB.

**Features:**
- Sidebar tree: Storage types > origins
- Cookie table with inline editing
- localStorage/sessionStorage key-value table
- IndexedDB database > store > records browser
- Clear storage actions

**CDP domains:** `Storage`, `DOMStorage`, `IndexedDB`, `CacheStorage`

**sumi-ui components:** TreeView, Table, PropertyList

### Panel 6: Security

Certificate and connection security overview.

**Features:**
- Security state summary (Secure / Not Secure)
- Certificate details
- TLS connection info
- Mixed content warnings

**CDP domains:** `Security`

**sumi-ui components:** PropertyList

## Global Features

### Tab Bar
Top-level tab switching between panels. Keyboard: number keys (1-6) or Ctrl+Left/Right.

### Command Palette
`Ctrl+P` opens a fuzzy-search command palette for quick access to any action across all panels.

### Connection Status
Status bar shows: connected target URL, page title, connection state.

### Keyboard Scheme
- `1`-`6` — switch panels
- `Ctrl+P` — command palette
- `Tab` / `Shift+Tab` — cycle focus between panes within a panel
- `Ctrl+C` / `q` — quit inspector
- Panel-specific shortcuts shown in status bar

## Iteration Plan

Elephant carpaccio slices — each delivers visible value:

### Phase 1: Console (MVP)
1. `hubcap inspect` launches TUI, connects to target, shows connection info
2. Console log streaming — messages appear as they arrive
3. Severity colouring and icons
4. REPL input — evaluate JS, show result
5. Object expansion in results (TreeView)
6. Filter bar — text filter
7. Filter bar — severity toggles
8. Error stack traces with source locations

### Phase 2: Elements
9. DOM tree — fetch and display document root
10. DOM tree — expand/collapse nodes, lazy child loading
11. Styles pane — matched CSS rules for selected node
12. Computed pane — resolved CSS values
13. Box model diagram
14. Live DOM mutation updates
15. Tab switching between Console and Elements

### Phase 3: Network
16. Network request table — capture and display requests
17. Request detail — Headers tab
18. Request detail — Response tab with preview
19. Timing waterfall bars
20. Type filter toggles
21. Text filter

### Phase 4: Sources + Application + Security
22. Source file tree
23. Source viewer
24. Breakpoint support (set/remove/hit)
25. Cookie table
26. localStorage/sessionStorage viewer
27. Security overview

## Dependencies

```
github.com/tomyan/sumi      — TUI framework
github.com/tomyan/sumi-ui   — component library
github.com/gorilla/websocket — existing hubcap dep for CDP
```

Both sumi and sumi-ui will use local `replace` directives during development.
