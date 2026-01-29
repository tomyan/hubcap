# hubcap tabs

List all open browser tabs and targets.

## When to use

Use `tabs` to discover which pages, service workers, and other targets are available in the connected browser. Use the `-target` global flag with any command to operate on a specific tab returned by this command.

## Usage

```
hubcap tabs
```

## Arguments

None.

## Flags

None.

## Output

Returns an array of target objects.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Target identifier used with the `-target` flag |
| type | string | Target type (e.g. `page`, `service_worker`, `background_page`) |
| title | string | Page or target title |
| url | string | URL loaded in the target |

```json
[
  {
    "id": "E3B0C44298FC1C14",
    "type": "page",
    "title": "Example Domain",
    "url": "https://example.com"
  },
  {
    "id": "A1B2C3D4E5F6G7H8",
    "type": "page",
    "title": "Google",
    "url": "https://www.google.com"
  }
]
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all tabs:

```
hubcap tabs
```

Get the URL of the first page-type tab:

```
hubcap tabs | jq -r '[.[] | select(.type=="page")][0].url'
```

Close every tab whose URL contains "ads" by chaining `tabs` into `close`:

```
for id in $(hubcap tabs | jq -r '.[] | select(.url | test("ads")) | .id'); do
  hubcap -target "$id" close
done
```

## See also

- [new](new.md) - open a new tab
- [close](close.md) - close a tab
- [version](version.md) - print browser version information
