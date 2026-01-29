# hubcap visible -- Check if an element is visible

## When to use

Check if an element is visible (not hidden by CSS). Use `exists` to check DOM presence regardless of visibility.

## Usage

```
hubcap visible <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector of the element to check |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| visible | boolean | Whether the element is visible |
| selector | string | The selector that was used |

```json
{"visible":true,"selector":".modal"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Check if a modal is visible:

```
hubcap visible '.modal'
```

Wait for an element, then check visibility:

```
hubcap wait '.toast' && hubcap visible '.toast' | jq -r '.visible'
```

## See also

- [exists](exists.md) - Check if an element exists in the DOM
- [count](count.md) - Count matching elements
- [bounds](bounds.md) - Get element bounding box
- [wait](wait.md) - Wait for an element to appear
