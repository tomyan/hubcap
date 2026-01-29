# hubcap close

Close the current browser tab.

## When to use

Use `close` to shut down the active tab or, with the `-target` global flag, a specific tab by its target ID. Use `tabs` first to discover available targets.

## Usage

```
hubcap close
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| closed | boolean | Whether the tab was successfully closed |
| targetId | string | The ID of the closed target |

```json
{
  "closed": true,
  "targetId": "E3B0C44298FC1C14"
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Close the current tab:

```
hubcap close
```

Close a specific tab by target ID:

```
hubcap -target E3B0C44298FC1C14 close
```

Open a page, take a screenshot, then close the tab:

```
hubcap new "https://example.com" && hubcap screenshot --output page.png && hubcap close
```

## See also

- [new](new.md) - open a new tab
- [tabs](tabs.md) - list open browser tabs
