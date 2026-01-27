# Elephant Carpaccio - CDP CLI

Thinnest possible vertical slices, each delivering demonstrable end-to-end value.

## Slice 1: Hello Chrome
**Goal**: Prove we can connect to Chrome and get a response.
```bash
cdp version
# Output: {"browser": "Chrome/120.0.0.0", "protocol": "1.3"}
```
- Connect to Chrome debug port
- Send Browser.getVersion
- Output JSON
- Exit with code 0 on success, 1 on failure

## Slice 2: List Tabs
**Goal**: Query Chrome state and output structured data.
```bash
cdp tabs
# Output: [{"id": "ABC", "title": "Google", "url": "https://google.com"}]
```
- Get all targets
- Filter to pages
- Output as JSON array

## Slice 3: Navigate
**Goal**: First mutation - change browser state.
```bash
cdp goto "https://example.com"
# Output: {"url": "https://example.com", "status": "loaded"}
```
- Navigate to URL
- Wait for load
- Return result

## Slice 4: Screenshot
**Goal**: Extract binary data from browser.
```bash
cdp screenshot --output page.png
# Creates: page.png
```
- Capture screenshot
- Write to file
- Support format flag (png/jpeg)

## Slice 5: Evaluate JS
**Goal**: Execute code in browser context.
```bash
cdp eval "document.title"
# Output: {"result": "Example Domain"}
```
- Evaluate expression
- Return serialized result
- Handle errors

## Slice 6: Query DOM
**Goal**: Find elements and return structured info.
```bash
cdp query "h1"
# Output: [{"nodeId": 5, "tagName": "H1", "text": "Example"}]
```
- querySelector
- Return node info
- Support --limit flag

## Slice 7: Click
**Goal**: First user interaction.
```bash
cdp click "button.submit"
# Output: {"clicked": true, "selector": "button.submit"}
```
- Find element
- Get coordinates
- Dispatch click

## Slice 8: Fill Input
**Goal**: Text input interaction.
```bash
cdp fill "#email" "test@example.com"
# Output: {"filled": true, "selector": "#email", "value": "test@example.com"}
```
- Focus element
- Clear existing
- Type text

## Slice 9: Console Messages
**Goal**: Subscribe to events and stream output.
```bash
cdp console --tail
# Streams NDJSON:
# {"level": "log", "text": "Hello"}
# {"level": "error", "text": "Oops"}
```
- Enable Console domain
- Stream events as NDJSON
- Ctrl+C to stop

## Slice 10: Network Watch
**Goal**: Monitor network activity.
```bash
cdp network watch
# Streams NDJSON:
# {"event": "request", "url": "...", "method": "GET"}
# {"event": "response", "url": "...", "status": 200}
```
- Enable Network domain
- Stream request/response events

## Slice 11: Get Cookies
**Goal**: Read storage data.
```bash
cdp cookies
# Output: [{"name": "session", "value": "abc", "domain": ".example.com"}]
```
- Get all cookies
- Support --domain filter

## Slice 12: Set Cookie
**Goal**: Write storage data.
```bash
cdp cookies set --name "test" --value "123" --domain "example.com"
# Output: {"success": true}
```
- Set cookie with options
- Verify success

## Slice 13: Self-Discovery
**Goal**: CLI describes its own capabilities.
```bash
cdp --list
# Output: ["version", "tabs", "goto", "screenshot", ...]

cdp goto --describe
# Output: {"command": "goto", "args": [{"name": "url", "required": true}], ...}
```
- List commands
- Describe each command

## Slice 14: Low-Level CDP Passthrough
**Goal**: Direct access to any CDP method.
```bash
cdp raw DOM.getDocument --depth 2
# Output: {"root": {"nodeId": 1, ...}}
```
- Parse domain.method
- Pass through args
- Return raw response

## Slice 15: Batch Operations
**Goal**: Multiple operations in one call.
```bash
echo '[{"cmd": "goto", "args": ["https://example.com"]}, {"cmd": "screenshot"}]' | cdp batch
# Output: [{"id": 0, "result": {...}}, {"id": 1, "result": {...}}]
```
- Read operations from stdin
- Execute sequentially
- Return all results

---

## Validation Checklist (before each slice)

- [ ] Is this the thinnest slice that delivers value?
- [ ] Does it build on previous slices?
- [ ] Is it demonstrable in isolation?
- [ ] What's the acceptance test?
- [ ] What could go wrong? (error cases)

---

## Current Slice: 1 - Hello Chrome

### Acceptance Test
```bash
# Start Chrome with: google-chrome --remote-debugging-port=9222
cdp version
# Should output JSON with browser version
# Exit code 0
```

### Error Cases
1. Chrome not running → exit 2, error message
2. Wrong port → exit 2, connection refused
3. Timeout → exit 3, timeout message

### Minimal Implementation
1. CLI entry point (main.go)
2. Connect to WebSocket
3. Send Browser.getVersion
4. Print response as JSON
5. Exit

Ready to begin?
