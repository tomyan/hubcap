# Inspector Layout Design

## Top Bar

```
● Google                          Console  Elements  Network  Sources
───────────────────────────────────────────────────────────────────────
```

**Left: Status Indicator**
- `●` green dot when connected, red when disconnected
- Page title next to the dot, truncated if long
- Click opens a detail overlay showing:
  - Full URL
  - Target ID
  - Connection status
  - Browser version
  - [Focus Tab] button — brings the Chrome tab to front
- Escape or click outside dismisses the overlay

**Right: Tab Bar**
- Tab labels right-aligned: `Console  Elements  Network  Sources`
- Active tab is bold/bright, others are dim
- Click to switch (future), number keys 1-4 to switch
- Only Console is functional initially, others show "Coming soon"

## Layout Structure

```
┌─ inspector.sumi ─────────────────────────────────────┐
│ <box class="topbar">                                  │
│   <box class="status">● {title}</box>                 │
│   <box class="tabs">                                  │
│     <text class="tab active">Console</text>           │
│     <text class="tab">Elements</text>                 │
│     <text class="tab">Network</text>                  │
│     <text class="tab">Sources</text>                  │
│   </box>                                              │
│ </box>                                                │
│ <box class="panel" flex-grow="1">                     │
│   {if activeTab.Get() == 0}                           │
│     <Console entries={entries} prompt={prompt}         │
│       cursor={cursor} />                              │
│   {/if}                                               │
│ </box>                                                │
└───────────────────────────────────────────────────────┘
```

## CSS

```css
.topbar { direction: row; border-bottom: single; }
.status { direction: row; dim: true; }
.status-dot-connected { color: green; }
.status-dot-disconnected { color: red; }
.tabs { direction: row; flex-grow: 1; justify: end; }
.tab { dim: true; padding: 0 1; }
.tab-active { bold: true; }
.panel { flex-grow: 1; }
```

## Status Overlay

When the status indicator is clicked (or a keyboard shortcut like `Ctrl+I`):

```
┌─ Connection ─────────────────────┐
│ URL:     https://www.google.com  │
│ Title:   Google                  │
│ Target:  0F4CDEF1F7ED...        │
│ Browser: Chrome 146.0.7680.165   │
│ Status:  ● Connected             │
│                                  │
│ [Focus Tab]         [Close: Esc] │
└──────────────────────────────────┘
```

This is a `position: fixed` overlay box centered on screen.

## Component Hierarchy

```
hubcap/cmd/hubcap/
  inspect.sumi          → Inspector shell (topbar + panel area)

sumi-ui/components/
  console/              → Console panel (log + prompt) — exists
  tabbar/               → Reusable tab bar component — new
```

The inspector shell is hubcap-specific. The tab bar is reusable (sumi-ui).

## Signals

```
activeTab    *sumi.Signal[int]      — 0=Console, 1=Elements, etc.
connected    *sumi.Signal[bool]     — connection status
title        *sumi.Signal[string]   — page title
```

## Keyboard

- `1-4` — switch tabs (when not typing in console prompt)
- `Ctrl+I` — toggle connection info overlay
- `Esc` — dismiss overlay

## Implementation Order

1. Top bar with status dot + title + tab labels (Console only active)
2. Console panel fills remaining space
3. Tab switching (dim/bold toggle, panel swap)
4. Status overlay on click/Ctrl+I
5. Focus Tab button in overlay
