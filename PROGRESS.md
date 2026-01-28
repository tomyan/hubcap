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

### Slice 53: Geolocation ✅
- `cdp geolocation <lat> <lon>` - set geolocation override

### Slice 54: Offline mode ✅
- `cdp offline <true|false>` - enable/disable offline mode

### Slice 55: Element screenshot ✅
- `cdp screenshot --output <file> --selector <css>` - capture just an element
- Returns bounding box with screenshot metadata

### Slice 56: Computed styles ✅
- `cdp styles <selector>` - get computed CSS styles for an element
- Returns common layout/styling properties

### Slice 57: Element layout ✅
- `cdp layout <selector> [--depth <n>]` - get comprehensive layout info
- Includes bounds, styles, and children with their layouts

### Slice 58: Response interception ✅
- `cdp intercept [--response] [--pattern <url>] [--replace old:new]` - intercept and modify responses
- `cdp intercept --disable` - disable interception
- Uses CDP Fetch domain for network interception
- Note: Interception requires persistent connection; works within single CDP session

### Slice 59: URL blocking ✅
- `cdp block <pattern>...` - block URLs matching patterns
- `cdp block --disable` - disable URL blocking
- Uses Network.setBlockedURLs CDP method

### Slice 60: Performance metrics ✅
- `cdp metrics` - get page performance metrics
- Returns timing metrics like Timestamp, Documents, Frames, JSEventListeners, etc.

### Slice 61: Accessibility tree ✅
- `cdp a11y` - get accessibility tree for the page
- Returns nodes with role, name, description, value, and properties

### Slice 62: Page source ✅
- `cdp source` - get full HTML source of the page

### Slice 63: Wait for network idle ✅
- `cdp waitidle [--idle <duration>]` - wait for network to be idle
- Default idle time: 500ms with no network activity

### Slice 64: Get all links ✅
- `cdp links` - get all links on the page
- Returns href and text for each anchor element

### Slice 65: File upload ✅
- `cdp upload <selector> <file>...` - upload files to a file input
- Uses DOM.setFileInputFiles CDP method

### Slice 66: Element exists ✅
- `cdp exists <selector>` - check if element exists (without waiting)
- Returns boolean `exists` field

### Slice 67: Wait for navigation ✅
- `cdp waitnav [--timeout <duration>]` - wait for navigation to complete
- Subscribes to Page.frameNavigated event

### Slice 68: Get input value ✅
- `cdp value <selector>` - get value of input/textarea/select
- Returns the current value property

### Slice 69: Wait for function ✅
- `cdp waitfn <expression> [--timeout <duration>]` - wait until JS is truthy
- Polls until expression evaluates to a truthy value

### Slice 70: List forms ✅
- `cdp forms` - get all forms with their inputs
- Returns form id, name, action, method, and inputs array

### Slice 71: Highlight element ✅
- `cdp highlight <selector>` - highlight element for debugging
- `cdp highlight --hide` - remove highlight
- Uses Overlay domain for visual highlighting

### Slice 72: List images ✅
- `cdp images` - get all images on the page
- Returns src, alt, width, height, loading for each image

### Slice 73-74: Scroll to edges ✅
- `cdp scrollbottom` - scroll to bottom of page
- `cdp scrolltop` - scroll to top of page

## Next Slices

## Test Command
```bash
# Run all tests (each package has its own Chrome instance)
go test -v ./...

# Run individual package tests
go test -v ./cmd/cdp
go test -v ./internal/cdp
```

## Commands Implemented (71 commands)
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
cdp screenshot --output <file> [--format png|jpeg|webp] [--quality 0-100] [--selector <css>]
cdp pdf --output <file> [--landscape] [--background]

# JavaScript execution
cdp eval <expression>

# DOM queries
cdp query <selector>
cdp html <selector>
cdp text <selector>
cdp attr <selector> <attribute>
cdp value <selector>
cdp count <selector>
cdp visible <selector>
cdp exists <selector>
cdp bounds <selector>
cdp styles <selector>
cdp layout <selector> [--depth <n>]
cdp forms
cdp images
cdp highlight <selector> [--hide]

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
cdp upload <selector> <file>...

# Keyboard input
cdp type <text>
cdp press <key>

# Scrolling
cdp scrollto <selector>
cdp scroll <x> <y>
cdp scrollbottom
cdp scrolltop

# Waiting
cdp wait <selector> [--timeout <duration>]
cdp waitload [--timeout <duration>]
cdp waitnav [--timeout <duration>]
cdp waitfn <expression> [--timeout <duration>]

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
cdp geolocation <latitude> <longitude>
cdp offline <true|false>

# Network interception
cdp intercept [--response] [--pattern <url>] [--replace old:new] [--disable]
cdp block <pattern>... [--disable]

# Performance & debugging
cdp metrics
cdp a11y
cdp source
cdp waitidle [--idle <duration>]
cdp links
```

## Known Issues / Deferred Items
- ~~Sessions not detached after use (minor resource leak)~~ **FIXED: Sessions now cached and detached on close**
- Navigate doesn't wait for actual load completion
- ~~Tests must run sequentially (-p 1) because they share Chrome instance~~ **FIXED: Each package now has its own Chrome instance**
- Special keys (Enter, Tab, etc.) not handled in type command
- No modifier key support (Ctrl, Alt, Shift) in type command
- ~~Long test runs can accumulate tabs causing Chrome memory pressure~~ **FIXED: Tests now use isolated tabs with proper cleanup**
