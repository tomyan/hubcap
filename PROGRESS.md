# CDP CLI Progress Tracker

## Current State
- **Last Commit**: c21ef75 - Slice 49 (Raw CDP support)
- **Chrome Status**: Running headless on port 9222

## Completed Slices

### Slice 1: Hello Chrome ✅
- `cdp version` - returns browser version info

### Slice 2: List Tabs ✅
- `cdp tabs` - lists page targets

### Slice 3: Navigate ✅
- `cdp goto <url>` - navigates first page to URL

### Slice 4: Screenshot ✅
- `cdp screenshot --output <file>` - capture page screenshot

### Slice 5: Evaluate JS ✅
- `cdp eval "<expression>"` - evaluate JavaScript in page context

### Slice 6: Query DOM ✅
- `cdp query "<selector>"` - return first matching element (nodeId, tagName, attributes)

### Slice 7: Click ✅
- `cdp click "<selector>"` - click first matching element

### Slice 8: Fill Input ✅
- `cdp fill "<selector>" "<text>"` - fill input field with text

### Slice 9: Get HTML ✅
- `cdp html "<selector>"` - get outer HTML of element

### Slice 10: Wait for Selector ✅
- `cdp wait "<selector>" [--timeout <duration>]` - wait for element to appear

### Slice 11: Get Text ✅
- `cdp text "<selector>"` - get innerText of element

### Slice 12: Type (keystroke by keystroke) ✅
- `cdp type "<text>"` - type text with individual key events
- Useful for inputs that need realistic typing (autocomplete, etc.)

### Slice 13: Console capture ✅
- `cdp console [--duration <d>]` - capture console messages
- Streams NDJSON output until duration expires
- Added event handling infrastructure to CDP client

### Slice 14: Cookie management ✅
- `cdp cookies` - list cookies
- `cdp cookies --set name=value [--domain <domain>]` - set cookie

### Slice 15: PDF export ✅
- `cdp pdf --output <file> [--landscape] [--background]` - export page as PDF

### Slice 16: Delete cookie ✅
- `cdp cookies --delete <name>` - delete a cookie by name

### Slice 17: Clear cookies ✅
- `cdp cookies --clear` - clear all cookies

### Slice 18: Focus element ✅
- `cdp focus <selector>` - focus on an element

### Slice 19: Network monitoring ✅
- `cdp network [--duration <d>]` - capture network requests/responses
- Streams NDJSON output with request/response events
- Includes URL, method, status, mimeType

### Slice 20: (Skipped - reserved for target selection)

### Slice 21: Press key ✅
- `cdp press <key>` - press a special key
- Supports: Enter, Tab, Escape, Backspace, Delete, Arrow keys, Home, End, PageUp, PageDown, Space

### Slice 22: Hover ✅
- `cdp hover <selector>` - hover over an element

### Slice 23: Get attribute ✅
- `cdp attr <selector> <name>` - get attribute value of an element

### Slice 24: Reload page ✅
- `cdp reload [--bypass-cache]` - reload the current page

### Slice 25: (Reserved for target selection)

### Slice 26: Back/Forward navigation ✅
- `cdp back` - navigate back in history
- `cdp forward` - navigate forward in history

### Slice 27: Get page title ✅
- `cdp title` - get the current page title

### Slice 28: Get page URL ✅
- `cdp url` - get the current page URL

### Slice 29: New tab ✅
- `cdp new [url]` - create a new tab, optionally navigate to URL

### Slice 30: Close tab ✅
- `cdp close` - close the current tab

### Slice 31: Double-click ✅
- `cdp dblclick <selector>` - double-click an element

### Slice 32: Right-click ✅
- `cdp rightclick <selector>` - right-click (context menu) an element

### Slice 33: Clear input ✅
- `cdp clear <selector>` - clear a text input field

### Slice 34: Select dropdown ✅
- `cdp select <selector> <value>` - select option by value

### Slice 35: Check checkbox ✅
- `cdp check <selector>` - check a checkbox

### Slice 36: Uncheck checkbox ✅
- `cdp uncheck <selector>` - uncheck a checkbox

### Slice 37: Scroll into view ✅
- `cdp scrollto <selector>` - scroll element into view

### Slice 38: Scroll by ✅
- `cdp scroll <x> <y>` - scroll by x,y pixels

### Slice 39: Count elements ✅
- `cdp count <selector>` - count matching elements

### Slice 40: Is visible ✅
- `cdp visible <selector>` - check if element is visible

### Slice 41: Bounding box ✅
- `cdp bounds <selector>` - get element position and size

### Slice 42: Wait for load ✅
- `cdp waitload [--timeout]` - wait for page load event

### Slice 43: Set viewport ✅
- `cdp viewport <width> <height>` - set browser viewport size

### Slice 44-46: Local storage ✅
- `cdp storage <key>` - get localStorage value
- `cdp storage <key> <value>` - set localStorage value
- `cdp storage --clear` - clear localStorage

### Slice 47: Handle dialog ✅
- `cdp dialog [accept|dismiss] [--text]` - handle alert/confirm/prompt dialogs

### Slice 48: Execute script file ✅
- `cdp run <file.js>` - execute JavaScript from file

### Slice 49: Raw CDP support ✅
- `cdp raw <method> [params-json]` - send CDP command to page
- `cdp raw --browser <method> [params-json]` - send to browser level

### Slice 50: Target selection ✅
- `--target <id|index>` global flag for page selection
- Accepts integer (0-based index) or target ID string
- All page-specific commands respect this flag

### Slice 51: Device emulation ✅
- `cdp emulate <device>` - emulate mobile devices
- Preset devices: iPhone 12, iPhone 12 Pro, iPhone 12 Pro Max, iPhone SE, Pixel 5, Galaxy S21, iPad, iPad Pro

### Slice 52: User agent ✅
- `cdp useragent <string>` - set custom user agent

## Next Slices

## Test Command
```bash
# Run tests sequentially (required because tests share Chrome instance)
go test -p 1 -v ./...

# Run individual package tests
go test -v ./cmd/cdp
go test -v ./internal/cdp
```

## Commands Implemented (50 commands)
```
# Browser info
cdp version
cdp tabs

# Navigation
cdp goto <url>
cdp back
cdp forward
cdp reload [--bypass-cache]

# Tab management
cdp new [url]
cdp close

# Page info
cdp title
cdp url

# Screenshots & PDF
cdp screenshot --output <file> [--format png|jpeg|webp] [--quality 0-100]
cdp pdf --output <file> [--landscape] [--background]

# JavaScript execution
cdp eval <expression>

# DOM queries
cdp query <selector>
cdp html <selector>
cdp text <selector>
cdp attr <selector> <attribute>
cdp count <selector>
cdp visible <selector>
cdp bounds <selector>

# Click actions
cdp click <selector>
cdp dblclick <selector>
cdp rightclick <selector>
cdp hover <selector>

# Form interactions
cdp fill <selector> <text>
cdp clear <selector>
cdp focus <selector>
cdp select <selector> <value>
cdp check <selector>
cdp uncheck <selector>

# Keyboard input
cdp type <text>
cdp press <key>

# Scrolling
cdp scrollto <selector>
cdp scroll <x> <y>

# Waiting
cdp wait <selector> [--timeout <duration>]
cdp waitload [--timeout <duration>]

# Viewport
cdp viewport <width> <height>

# Cookies
cdp cookies [--set name=value] [--domain <domain>] [--delete <name>] [--clear]

# Local storage
cdp storage <key> [value] [--clear]

# Streaming/monitoring
cdp console [--duration <duration>]
cdp network [--duration <duration>]

# Dialog handling
cdp dialog [accept|dismiss] [--text <prompt>]

# Script execution
cdp run <file.js>

# Device emulation
cdp emulate <device>
cdp useragent <string>
```

## Known Issues / Deferred Items
- Sessions not detached after use (minor resource leak)
- No --target flag for page selection (always uses first page)
- Navigate doesn't wait for actual load completion
- Tests must run sequentially (-p 1) because they share Chrome instance
- Special keys (Enter, Tab, etc.) not handled in type command
- No modifier key support (Ctrl, Alt, Shift) in type command
