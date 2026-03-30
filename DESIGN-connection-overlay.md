# Connection Overlay Design

## Overview

A modal overlay accessed via the status dot in the inspector topbar. Shows connection status, available Chrome tabs, and controls for switching targets, focusing the browser, and creating new tabs.

## Behaviour

### Visibility Rules
- **Connected**: overlay hidden by default. Click dot or `Ctrl+I` toggles.
- **Disconnected**: overlay auto-shows with reconnecting status. User can dismiss with `Esc` to see the console at the point of disconnection. Dot stays red. `Ctrl+I` re-shows.
- **On reconnect**: overlay auto-hides, dot goes green.

### Layout

```
┌─ Connection ──────────────────────────────────────────┐
│                                                        │
│  ● Connected to Google                                 │
│  https://www.google.com                                │
│  Target: 0F4CDEF...  Browser: Chrome 146.0.7680.165   │
│                                                        │
│  Tabs ─────────────────────────────────────────────    │
│  ❯ _                                                   │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄    │
│  → Google                    https://google.com        │
│    GitHub                    https://github.com        │
│    localhost:3000            http://localhost:3000      │
│                                                        │
│  f Focus Tab  n New Tab  Enter Switch        Esc Close │
└────────────────────────────────────────────────────────┘
```

- `→` marks the currently connected tab
- Highlighted tab shown with bright/bold
- Arrow keys move selection, Enter switches target
- Find-as-you-type filters the tab list

### Disconnected State

```
┌─ Connection ──────────────────────────────────────────┐
│                                                        │
│  ● Disconnected — reconnecting every 2s               │
│  Last: Google — https://www.google.com                 │
│                                                        │
│  Tabs ─────────────────────────────────────────────    │
│  ❯ _                                                   │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄    │
│  (no tabs available)                                   │
│                                                        │
│                                             Esc Close  │
└────────────────────────────────────────────────────────┘
```

## Component Architecture

### Files

```
hubcap/internal/inspector/
  inspector.sumi          — shell (topbar + panel)
  overlay.sumi            — connection overlay (NEW)
  overlay_data.go         — OverlayData type + helpers (NEW)
```

### overlay.sumi

A `position: fixed` box centered on screen, with:
- Connection status section
- Tab list with find-as-you-type filter
- Keyboard shortcut hints

```sumi
<style>
.overlay { position: fixed; border: single; padding: 1 2; }
.overlay-title { bold: true; }
.status-connected { color: #50fa7b; bold: true; }
.status-disconnected { color: #ff5555; bold: true; }
.url { dim: true; }
.meta { dim: true; }
.tab-list { overflow: auto; }
.tab { padding: 0 1; }
.tab:hover { color: white; }
.tab-current { color: #50fa7b; }
.tab-selected { bold: true; color: white; }
.tab-title { }
.tab-url { dim: true; }
.hints { dim: true; border-top: single; padding: 1 0 0 0; }
</style>

<script>
var visible *sumi.Signal[bool]
var connected *sumi.Signal[bool]
var pageTitle *sumi.Signal[string]
var pageURL *sumi.Signal[string]
var targetID *sumi.Signal[string]
var browserVersion *sumi.Signal[string]
var tabs *sumi.Signal[[]TabInfo]
var selectedIdx *sumi.Signal[int]
var currentTargetID *sumi.Signal[string]
var filter *sumi.Signal[string]
</script>

<box class="overlay">
    ...
</box>
```

### overlay_data.go

```go
package inspector

// TabInfo represents a Chrome tab in the overlay list.
type TabInfo struct {
    ID    string
    Title string
    URL   string
}
```

### Signals Flow

```
inspectSession (Go)
  ├── connected signal      → overlay status display
  ├── pageTitle signal      → overlay + topbar
  ├── pageURL signal        → overlay
  ├── targetID signal       → overlay (current marker)
  ├── browserVersion signal → overlay
  ├── tabs signal           → overlay tab list (refreshed periodically)
  └── overlayVisible signal → show/hide overlay

inspector.sumi
  ├── topbar (dot click → toggle overlay)
  ├── panel (console)
  └── <overlay.Overlay ... /> (position: fixed, conditional)
```

### Tab List Refresh

The `inspectSession` periodically fetches the tab list (every 2s when overlay is visible, or on overlay open):

```go
func (s *inspectSession) refreshTabs(ctx context.Context) {
    pages, err := s.client.Pages(ctx)
    if err != nil { return }
    tabInfos := make([]TabInfo, len(pages))
    for i, p := range pages {
        tabInfos[i] = TabInfo{ID: p.ID, Title: p.Title, URL: p.URL}
    }
    s.app.Do(func() { s.tabs.Set(tabInfos) })
}
```

### Keyboard Handling

When overlay is visible, it captures all keyboard input:

| Key | Action |
|-----|--------|
| `Esc` | Close overlay |
| `↑`/`↓` | Move tab selection |
| `Enter` | Switch to selected tab |
| `f` | Focus Chrome window + selected tab |
| `n` | New tab (prompts for URL or opens blank) |
| Printable chars | Filter tab list |
| `Backspace` | Delete filter char |
| `Ctrl+I` | Close overlay |

### Target Switching

When the user presses Enter on a tab:

1. Stop current console capture
2. Update targetID to new tab
3. Restart console capture on new target
4. Update pageTitle, pageURL
5. Close overlay

```go
func (s *inspectSession) switchTarget(ctx context.Context, newTargetID string) {
    // Stop old capture
    s.mu.Lock()
    if s.stopCapture != nil { s.stopCapture() }
    s.mu.Unlock()

    // Start new capture
    s.targetID = newTargetID
    messages, stop, err := s.client.CaptureConsole(ctx, newTargetID)
    ...
    // Update signals
    s.app.Do(func() {
        s.currentTargetID.Set(newTargetID)
        s.pageTitle.Set(title)
        s.pageURL.Set(url)
    })
}
```

### Focus Tab

Uses Chrome's `Target.activateTarget` CDP method to bring a tab to front:

```go
func (s *inspectSession) focusTab(ctx context.Context, targetID string) {
    s.client.Call(ctx, "Target.activateTarget", map[string]interface{}{
        "targetId": targetID,
    })
    // Also focus the Chrome window via CDP if possible
}
```

### New Tab

```go
func (s *inspectSession) newTab(ctx context.Context, url string) {
    if url == "" { url = "about:blank" }
    s.client.Call(ctx, "Target.createTarget", map[string]interface{}{
        "url": url,
    })
    s.refreshTabs(ctx)
}
```

## Implementation Plan — Elephant Carpaccio

Each slice is a working increment:

### Slice 1: Overlay skeleton
- `overlay.sumi` with position: fixed, border, static content
- `overlayVisible` signal toggled by `Ctrl+I` in inspector
- `Esc` closes overlay
- No tab list yet, just connection status

### Slice 2: Connection info
- Pass connection signals to overlay (pageTitle, pageURL, targetID, browserVersion, connected)
- Display current status with green/red dot
- Disconnected state shows "Reconnecting..."

### Slice 3: Tab list display
- `inspectSession.refreshTabs()` fetches tabs on overlay open
- Display tab list with current tab marked
- Arrow keys move selection

### Slice 4: Find-as-you-type
- Filter input at top of tab list
- Tab list filters as user types
- `edit.State` for filter input

### Slice 5: Switch target
- Enter on selected tab switches the CDP target
- Console capture restarts on new target
- Overlay closes after switch

### Slice 6: Focus tab
- `f` key activates the selected tab in Chrome
- Uses `Target.activateTarget`

### Slice 7: New tab
- `n` key creates a new Chrome tab
- Tab list refreshes to include it
- Optional: prompt for URL

### Slice 8: Auto-show on disconnect
- Overlay auto-shows when connection drops
- Shows reconnecting status
- Auto-hides on reconnect

## Dependencies

### Sumi features needed (all exist):
- `position: fixed` — overlay positioning
- `overflow: auto` — scrollable tab list
- `:hover` — tab hover highlighting
- `{if}` / `{for}` — conditional rendering and tab list
- `border: single` — overlay border
- CSS inheritance — styles cascade to text

### Sumi features that would help but aren't blocking:
- `onclick` handlers on elements — use keyboard shortcuts instead
- `z-index` — overlay is position:fixed, renders on top already
- Focus management — overlay captures all input when visible

### Hubcap CDP methods needed:
- `Target.getTargets` — list tabs (already used by `Pages()`)
- `Target.activateTarget` — focus a tab
- `Target.createTarget` — new tab
- `Target.setDiscoverTargets` — receive targetDestroyed events (already added)
