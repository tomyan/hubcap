# Changelog

## v1.7.0

- `eval` and `run` now automatically await promises — expressions that return a Promise resolve before returning the result
- Added links to docs site and blog post in README

## v1.6.0

- Added `--wait` flag to `new` command — creates a tab and waits for the page to fully load before returning

## v1.5.1

- Fixed full-page screenshots capturing excess whitespace — now uses CSS content dimensions instead of device pixel dimensions

## v1.5.0

- Added `--full` flag to `screenshot` command — captures the entire scrollable page, not just the viewport
