# hubcap new

Open a new browser tab.

## When to use

Use `new` to open a new browser tab, optionally navigating to a URL. Use the `-target` global flag to interact with the new tab afterward.

## Usage

```
hubcap new [url]
```

## Arguments

| Argument | Type   | Required | Description                                      |
|----------|--------|----------|--------------------------------------------------|
| url      | string | No       | URL to open in new tab (defaults to about:blank)  |

## Flags

None.

## Output

| Field    | Type   | Description                          |
|----------|--------|--------------------------------------|
| targetId | string | Unique identifier for the new tab    |
| url      | string | The URL opened in the new tab        |

```json
{"targetId":"A1B2C3D4E5F6","url":"about:blank"}
```

## Errors

| Condition            | Exit code | Stderr                        |
|----------------------|-----------|-------------------------------|
| Chrome not connected | 2         | `error: connecting to Chrome: ...` |
| Tab creation timeout | 3         | `error: timeout`              |

## Examples

Open a blank new tab:

```
hubcap new
```

Open a new tab and navigate to a URL:

```
hubcap new https://example.com
```

Open a tab and capture its ID for later use (chaining):

```
TAB_ID=$(hubcap new https://example.com | jq -r '.targetId') && hubcap -target "$TAB_ID" title
```

## See also

- [close](close.md) - Close a browser tab
- [tabs](tabs.md) - List open browser tabs
- [goto](goto.md) - Navigate an existing tab to a URL
