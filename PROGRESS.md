# CDP CLI Progress Tracker

## Current State
- **Last Commit**: d2cf97e - Slice 3 (Navigate)
- **In Progress**: Slice 4 (Screenshot) - GREEN phase complete, need CLI command
- **Chrome Status**: Running headless on port 9222

## Completed Slices

### Slice 1: Hello Chrome ✅
- `cdp version` - returns browser version info
- CDP client with WebSocket connection
- JSON/NDJSON output formats
- Environment variable support (CDP_PORT, CDP_HOST)
- Semantic exit codes

### Slice 2: List Tabs ✅
- `cdp tabs` - lists page targets
- Targets() and Pages() methods
- withClient() helper to reduce duplication

### Slice 3: Navigate ✅
- `cdp goto <url>` - navigates first page to URL
- Navigate() method with target attachment
- CallSession() for session-scoped commands

### Slice 4: Screenshot 🔄 (In Progress)
- Screenshot() method implemented and tested
- Returns PNG/JPEG binary data
- **TODO**: Add CLI command with --output flag

## Deferred Issues (Adversarial Reviews)

### From Slice 1
- [ ] Events not handled in readMessages (only responses with ID > 0)
- [ ] No reconnection if connection drops
- [ ] Context not propagated to HTTP client timeout
- [ ] No connection pooling

### From Slice 2
- [ ] Test created about:blank target but didn't clean it up

### From Slice 3
- [ ] No --target flag to select which page to navigate
- [ ] Navigate doesn't wait for actual load completion (TODO in code)
- [ ] Session not detached after use (resource leak)
- [ ] CallSession duplicates Call code structure

### From Slice 4 (Current)
- [ ] Session not detached after screenshot
- [ ] No --selector flag for element screenshots
- [ ] No --full-page flag for full page capture

## Architecture Notes

```
cmd/cdp/main.go          - CLI entry point, flag parsing, command dispatch
internal/cdp/client.go   - CDP WebSocket client, all CDP methods
internal/cdp/client_test.go - Integration tests (require Chrome on :9222)
```

Key patterns:
- `withClient()` - wraps connection/error handling for CLI commands
- `Call()` - browser-level CDP commands
- `CallSession()` - target session-scoped CDP commands
- Tests use `testing.Short()` to skip integration tests

## Test Commands
```bash
# Start Chrome for testing
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --remote-debugging-port=9222 --headless=new &

# Run all tests
go test -v ./...

# Run short tests only (no Chrome needed)
go test -short ./...

# Build CLI
go build -o /tmp/cdp ./cmd/cdp
```

## Next Steps

1. **Complete Slice 4**: Add screenshot CLI command with --output, --format flags
2. **Slice 5**: Evaluate JS (`cdp eval "<expression>"`)
3. **Slice 6**: Query DOM (`cdp query "<selector>"`)
4. **Slice 7**: Click (`cdp click "<selector>"`)
5. **Slice 8**: Fill input (`cdp fill "<selector>" "<text>"`)

## File Structure
```
/Users/tom/projects/cdp-cli/
├── DESIGN.md           # Full design document
├── SLICES.md           # Elephant carpaccio breakdown
├── PROGRESS.md         # This file - progress tracking
├── cmd/cdp/
│   ├── main.go         # CLI
│   └── main_test.go    # CLI tests
├── internal/cdp/
│   ├── client.go       # CDP client
│   └── client_test.go  # Client tests
├── go.mod
└── go.sum
```

## Resume Instructions

To continue from this checkpoint:
1. Chrome should be running: `curl http://localhost:9222/json/version`
2. Complete Slice 4 by adding CLI command for screenshot
3. Run tests: `go test -v ./...`
4. Commit and proceed to Slice 5
