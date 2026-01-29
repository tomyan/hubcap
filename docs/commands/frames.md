# hubcap frames

List all frames and iframes on the current page.

## When to use

Use `frames` to discover all frames and iframes embedded in the page. Use the returned frame IDs with `evalframe` to execute JavaScript in a specific frame context.

## Usage

```
hubcap frames
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| frames | array | Array of frame objects |
| frames[].id | string | Frame identifier for use with `evalframe` |
| frames[].url | string | URL loaded in the frame |
| frames[].name | string | The frame's `name` attribute |
| frames[].parentId | string | Parent frame identifier (empty for main frame) |
| count | number | Total number of frames found |

```json
{
  "frames": [
    {
      "id": "F1A2B3C4D5E6",
      "url": "https://example.com/widget",
      "name": "ad-frame",
      "parentId": "MAIN_FRAME_ID"
    }
  ],
  "count": 1
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all frames:

```
hubcap frames
```

Get the URL of each iframe:

```
hubcap frames | jq -r '.frames[].url'
```

Run JavaScript inside a specific frame:

```
FRAME_ID=$(hubcap frames | jq -r '.frames[0].id')
hubcap evalframe "$FRAME_ID" "document.title"
```

## See also

- [evalframe](evalframe.md) - execute JavaScript in a specific frame
- [source](source.md) - get the full HTML source of the main document
