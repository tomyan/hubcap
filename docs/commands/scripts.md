# hubcap scripts

List all script elements on the current page.

## When to use

Use `scripts` to enumerate every `<script>` tag on the page, including src, type, and whether it loads asynchronously. Useful for auditing which JavaScript files are loaded or detecting unwanted third-party scripts.

## Usage

```
hubcap scripts
```

## Arguments

None.

## Flags

None.

## Output

Returns an object containing an array of script info objects.

| Field | Type | Description |
|-------|------|-------------|
| scripts | array | Array of script info objects |
| scripts[].src | string | The `src` attribute, empty for inline scripts |
| scripts[].type | string | The `type` attribute (e.g. `module`, `text/javascript`) |
| scripts[].async | boolean | Whether the script has the `async` attribute |
| scripts[].defer | boolean | Whether the script has the `defer` attribute |
| scripts[].inline | boolean | Whether the script is inline (has no `src`) |

```json
{"scripts":[{"src":"https://example.com/app.js","type":"module","async":false,"defer":false,"inline":false},{"src":"","type":"text/javascript","async":false,"defer":false,"inline":true}]}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all scripts:

```
hubcap scripts
```

Count the number of external scripts:

```
hubcap scripts | jq '[.scripts[] | select(.src != "")] | length'
```

Detect third-party scripts by filtering out same-origin sources:

```
ORIGIN=$(hubcap url | jq -r '.url' | sed 's|^\(https\?://[^/]*\).*|\1|')
hubcap scripts | jq --arg o "$ORIGIN" '[.scripts[] | select(.src != "" and (.src | startswith($o) | not))]'
```

## See also

- [links](links.md) - extract all links from the page
- [images](images.md) - list all images on the page
- [coverage](coverage.md) - get JavaScript code coverage
