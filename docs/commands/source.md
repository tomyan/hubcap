# hubcap source

Get the full HTML source of the current page.

## When to use

Use `source` to retrieve the complete outer HTML of the document. Use `html <selector>` to get the outer HTML of a specific element instead of the entire page.

## Usage

```
hubcap source
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| html | string | The full HTML source of the document |

```json
{
  "html": "<!DOCTYPE html><html><head><title>Example</title></head><body><h1>Hello</h1></body></html>"
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Get the page source:

```
hubcap source
```

Save the page HTML to a file:

```
hubcap source | jq -r '.html' > page.html
```

Check whether the page contains a specific string:

```
hubcap source | jq -r '.html' | grep -q "viewport" && echo "Has viewport meta"
```

Pipe the source into an external validator:

```
hubcap source | jq -r '.html' | html-validate -
```

## See also

- [html](html.md) - get outer HTML of a specific element
- [info](info.md) - get combined page information
