# Changelog

## v1.8.0

- New `bridge` command — persistent bidirectional message channel between CLI and client-side JavaScript via LDJSON over stdio
- Supports inline scripts and `--file` flag, async iterator for receiving messages, `send()` for outbound messages
- Keepalive heartbeat ensures JS cleanup when the hubcap process exits
- Multiple bridges can coexist in the same tab via random instance isolation
- Fixed `PressKey` sending `Unidentified` key events by adding `code` and `text` fields to CDP key dispatch
- Fixed flaky tests across the suite by replacing `NewTab` + `time.Sleep` with `NewTabAndWait`

## v1.7.0

- `eval` and `run` now automatically await promises — expressions that return a Promise resolve before returning the result
- Added links to docs site and blog post in README

## v1.6.0

- Added `--wait` flag to `new` command — creates a tab and waits for the page to fully load before returning

## v1.5.1

- Fixed full-page screenshots capturing excess whitespace — now uses CSS content dimensions instead of device pixel dimensions

## v1.5.0

- Added `--full` flag to `screenshot` command — captures the entire scrollable page, not just the viewport
