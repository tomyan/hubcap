# hubcap command reference

Complete reference for all hubcap commands, organized by task.

## Global flags

All commands accept these flags before the command name:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-port <n>` | int | `9222` / `HUBCAP_PORT` | Chrome debug port |
| `-host <s>` | string | `localhost` / `HUBCAP_HOST` | Chrome debug host |
| `-timeout <d>` | duration | `10s` | Command timeout |
| `-output <fmt>` | string | `json` | Output format: `json`, `ndjson`, `text` |
| `-quiet` | bool | `false` | Suppress non-essential output |
| `-target <id>` | string | first page | Target page by index or ID |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (element not found, invalid args, protocol error) |
| 2 | Chrome connection failed |
| 3 | Timeout exceeded |

---

## Navigate & manage tabs

| Task | Command | Notes |
|------|---------|-------|
| Open URL | `goto <url>` | Add `--wait` to block until loaded |
| Go back | `back` | |
| Go forward | `forward` | |
| Reload page | `reload` | `--bypass-cache` to skip cache |
| Open new tab | `new [url]` | Returns `targetId` |
| Close current tab | `close` | |
| List tabs | `tabs` | Use `-target <id>` to switch |
| Wait for page load | `waitload` | `--timeout 30s` default |
| Wait for navigation | `waitnav` | `--timeout 30s` default |
| Wait for URL match | `waiturl <pattern>` | `--timeout 30s` default |
| Wait for network idle | `waitidle` | `--idle 500ms` default |

## Read page info

| Task | Command | Notes |
|------|---------|-------|
| Get page title | `title` | |
| Get page URL | `url` | |
| Get combined info | `info` | Title + URL + meta in one call |
| Get full HTML source | `source` | |
| Get meta tags | `meta` | |
| Get all links | `links` | Returns `href` + `text` |
| Get all images | `images` | |
| Get all scripts | `scripts` | |
| Get all tables | `tables` | Extracts headers + rows |
| Get all forms | `forms` | Includes input fields |
| List frames/iframes | `frames` | Returns frame IDs for `evalframe` |

## Query DOM elements

| Task | Command | Notes |
|------|---------|-------|
| Query element | `query <sel>` | Returns nodeId, tagName, attributes |
| Get outer HTML | `html <sel>` | |
| Get inner text | `text <sel>` | |
| Get attribute | `attr <sel> <name>` | |
| Get input value | `value <sel>` | For input/textarea/select |
| Count matches | `count <sel>` | Returns integer count |
| Check visibility | `visible <sel>` | Returns boolean |
| Check existence | `exists <sel>` | Returns boolean, never errors on missing |
| Get bounding box | `bounds <sel>` | Returns x, y, width, height |
| Get all CSS styles | `styles <sel>` | All computed properties |
| Get one CSS property | `computed <sel> <prop>` | Single computed value |
| Get layout + children | `layout <sel>` | `--depth N` controls child depth |
| Query shadow DOM | `shadow <host> <inner>` | Two selectors: host, then inner |
| Find text on page | `find <text>` | Returns match count + positions |
| Get selected text | `selection` | Current document selection |
| Get caret position | `caret <sel>` | Cursor offset in input |
| List event listeners | `listeners <sel>` | |

## Click & interact

| Task | Command | Notes |
|------|---------|-------|
| Click element | `click <sel>` | |
| Double-click | `dblclick <sel>` | |
| Right-click | `rightclick <sel>` | Context menu trigger |
| Triple-click | `tripleclick <sel>` | Selects paragraph |
| Click at coordinates | `clickat <x> <y>` | Float coordinates |
| Hover element | `hover <sel>` | Triggers `:hover` styles |
| Move mouse | `mouse <x> <y>` | Moves without clicking |
| Drag and drop | `drag <src> <dest>` | Two CSS selectors |

## Touch gestures

| Task | Command | Notes |
|------|---------|-------|
| Tap element | `tap <sel>` | Touch tap for mobile |
| Swipe gesture | `swipe <sel> <dir>` | `left`, `right`, `up`, `down` |
| Pinch zoom | `pinch <sel> <dir>` | `in` or `out` |

## Form input

| Task | Command | Notes |
|------|---------|-------|
| Fill input (clear + type) | `fill <sel> <text>` | Clears first, then types |
| Clear input | `clear <sel>` | |
| Type keystrokes | `type <text>` | Types into focused element |
| Press key combo | `press <key>` | e.g. `Enter`, `Ctrl+a`, `Ctrl+Shift+n` |
| Focus element | `focus <sel>` | |
| Select dropdown | `select <sel> <value>` | By option value |
| Check checkbox | `check <sel>` | |
| Uncheck checkbox | `uncheck <sel>` | |
| Set value directly | `setvalue <sel> <val>` | Bypasses input events |
| Upload files | `upload <sel> <file>...` | File input selector + paths |
| Dispatch event | `dispatch <sel> <type>` | Custom DOM event |

## Scroll

| Task | Command | Notes |
|------|---------|-------|
| Scroll by pixels | `scroll <x> <y>` | Relative scroll |
| Scroll to element | `scrollto <sel>` | scrollIntoView |
| Scroll to top | `scrolltop` | |
| Scroll to bottom | `scrollbottom` | |

## Wait for conditions

| Task | Command | Notes |
|------|---------|-------|
| Wait for element | `wait <sel>` | `--timeout 30s` default |
| Wait for text | `waittext <text>` | `--timeout 30s` default |
| Wait for removal | `waitgone <sel>` | `--timeout 30s` default |
| Wait for JS truthy | `waitfn <expr>` | `--timeout 30s` default |
| Wait for page load | `waitload` | `--timeout 30s` default |
| Wait for navigation | `waitnav` | `--timeout 30s` default |
| Wait for URL match | `waiturl <pattern>` | `--timeout 30s` default |
| Wait for network idle | `waitidle` | `--idle 500ms` default |
| Wait for request | `waitrequest <pattern>` | `--timeout 30s` default |
| Wait for response | `waitresponse <pattern>` | `--timeout 30s` default |

## Screenshots & export

| Task | Command | Notes |
|------|---------|-------|
| Screenshot page | `screenshot --output f.png` | `--format`, `--quality`, `--selector`, `--base64` |
| Export PDF | `pdf --output f.pdf` | `--landscape`, `--background` |

## Cookies & storage

| Task | Command | Notes |
|------|---------|-------|
| List cookies | `cookies` | No flags = list all |
| Set cookie | `cookies --set k=v` | Optional `--domain` |
| Delete cookie | `cookies --delete <name>` | Optional `--domain` |
| Clear cookies | `cookies --clear` | |
| Get localStorage | `storage <key>` | |
| Set localStorage | `storage <key> <val>` | |
| Clear localStorage | `storage --clear` | |
| Get sessionStorage | `session <key>` | |
| Set sessionStorage | `session <key> <val>` | |
| Clear sessionStorage | `session --clear` | |
| Read clipboard | `clipboard --read` | |
| Write clipboard | `clipboard --write <text>` | |

## Network monitoring

| Task | Command | Notes |
|------|---------|-------|
| Stream network events | `network` | NDJSON; `--duration` optional |
| Capture HAR | `har` | `--duration 5s` default |
| Get response body | `responsebody <id>` | Use requestId from network/har |
| Intercept requests | `intercept` | `--pattern`, `--replace`, `--response` |
| Disable intercept | `intercept --disable` | |
| Block URLs | `block <pattern>...` | |
| Unblock URLs | `block --disable` | |
| Throttle network | `throttle <preset>` | `3g`, `slow3g`, etc. |
| Disable throttle | `throttle --disable` | |

## Device emulation

| Task | Command | Notes |
|------|---------|-------|
| Emulate device | `emulate <device>` | e.g. `iPhone-12`, `Pixel-5` |
| Set user agent | `useragent <string>` | |
| Set geolocation | `geolocation <lat> <lon>` | |
| Set offline mode | `offline <true\|false>` | |
| Emulate CSS media | `media` | `--color-scheme`, `--reduced-motion`, `--forced-colors` |
| Set viewport | `viewport <w> <h>` | |
| Set permission | `permission <name> <state>` | `granted`, `denied`, `prompt` |

## Monitoring

| Task | Command | Notes |
|------|---------|-------|
| Stream console | `console` | NDJSON; `--duration` optional |
| Stream JS errors | `errors` | NDJSON; `--duration` optional |

## Analysis

| Task | Command | Notes |
|------|---------|-------|
| Performance metrics | `metrics` | JS heap, DOM nodes, layout count |
| Accessibility tree | `a11y` | |
| JS code coverage | `coverage` | |
| CSS rule coverage | `csscoverage` | |
| List stylesheets | `stylesheets` | |
| DOM snapshot | `domsnapshot` | |

## Profiling

| Task | Command | Notes |
|------|---------|-------|
| Heap snapshot | `heapsnapshot --output <f>` | V8 heap; open in DevTools Memory |
| Performance trace | `trace --output <f>` | `--duration 1s` default; open in DevTools Performance |

## JavaScript

| Task | Command | Notes |
|------|---------|-------|
| Eval expression | `eval <expr>` | Returns typed value |
| Eval in frame | `evalframe <frameId> <expr>` | Use `frames` to get IDs |
| Run JS file | `run <file.js>` | Reads file, evaluates contents |

## Advanced

| Task | Command | Notes |
|------|---------|-------|
| Raw protocol command | `raw <method> [json]` | `--browser` for browser-level |
| Handle dialog | `dialog <accept\|dismiss>` | `--text` for prompt input |
| Highlight element | `highlight <sel>` | `--hide` to remove |
| Browser version | `version` | |

---

## Common patterns

Navigate and interact:

```
hubcap goto --wait https://example.com && hubcap click '#login'
```

Fill a form:

```
hubcap fill '#email' 'user@test.com' && hubcap fill '#password' 'secret' && hubcap click '#submit'
```

Wait then read:

```
hubcap wait '.results' && hubcap text '.results'
```

Screenshot after load:

```
hubcap goto --wait https://example.com && hubcap screenshot --output page.png
```

Monitor network during action:

```
hubcap network --duration 5s &
hubcap click '#fetch-btn'
wait
```

Conditional check:

```
hubcap exists '.error' && hubcap text '.error'
```

Extract structured data:

```
hubcap tables | jq '.tables[0].rows'
```

Mobile emulation flow:

```
hubcap emulate iPhone-12 && hubcap goto --wait https://example.com && hubcap screenshot --output mobile.png
```
