# hubcap goto

Navigate the browser to a URL.

## When to use

Use `goto` to navigate to a URL in the current tab. Add `--wait` to block until the page is fully loaded. Use `reload` to reload the current page instead, or `back`/`forward` for history navigation.

## Usage

```
hubcap goto [--wait] <url>
```

## Arguments

| Argument | Type   | Required | Description            |
|----------|--------|----------|------------------------|
| url      | string | Yes      | The URL to navigate to |

## Flags

| Flag   | Type | Default | Description                    |
|--------|------|---------|--------------------------------|
| --wait | bool | false   | Wait for page load to complete |

## Output

Without `--wait`:

| Field    | Type   | Description          |
|----------|--------|----------------------|
| frameId  | string | Frame identifier     |
| loaderId | string | Loader identifier    |
| url      | string | The URL navigated to |

```json
{"frameId":"ABC123","loaderId":"DEF456","url":"https://example.com"}
```

With `--wait`:

| Field    | Type   | Description              |
|----------|--------|--------------------------|
| url      | string | The URL navigated to     |
| frameId  | string | Frame identifier         |
| loaderId | string | Loader identifier        |
| loaded   | bool   | Whether the page loaded  |

```json
{"url":"https://example.com","frameId":"ABC123","loaderId":"DEF456","loaded":true}
```

## Errors

| Condition            | Exit code | Stderr                         |
|----------------------|-----------|--------------------------------|
| Missing URL argument | 1         | `error: url argument required` |
| Chrome not connected | 2         | `error: chrome not connected`  |
| Navigation timeout   | 3         | `error: navigation timeout`    |

## Examples

Navigate to a URL:

```
hubcap goto https://example.com
```

Navigate and wait for load:

```
hubcap goto --wait https://example.com
```

Navigate, wait, then take a screenshot (chaining):

```
hubcap goto --wait https://example.com && hubcap screenshot --output page.png
```

Navigate and extract the page title:

```
hubcap goto --wait https://example.com && hubcap title
```

## See also

- [reload](reload.md) - Reload the current page
- [back](back.md) - Navigate back in history
- [forward](forward.md) - Navigate forward in history
- [url](url.md) - Get the current page URL
- [waitload](waitload.md) - Wait for the page load event
