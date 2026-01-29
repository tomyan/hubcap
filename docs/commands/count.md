# hubcap count -- Count elements matching a selector

## When to use

Count elements matching a selector. Returns 0 if none found (does not error). Use `exists` for a boolean check.

## Usage

```
hubcap count <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| selector | string | yes | CSS selector to count matches for |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| count | number | The number of matching elements |
| selector | string | The selector that was used |

```json
{"count":5,"selector":"ul.results li"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Count list items:

```
hubcap count 'ul.results li'
```

Count items and use the result in a script:

```
hubcap count '.notification' | jq -r '.count'
```

## See also

- [exists](exists.md) - Check if an element exists
- [query](query.md) - Query a DOM element
- [visible](visible.md) - Check if an element is visible
