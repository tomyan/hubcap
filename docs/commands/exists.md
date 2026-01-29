# hubcap exists -- Check if an element exists in the DOM

## When to use

Check if an element exists in the DOM. Never errors on missing elements -- returns `false` instead. Use `visible` to also check CSS visibility.

## Usage

```
hubcap exists <selector>
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
| exists | boolean | Whether the element exists in the DOM |
| selector | string | The selector that was used |

```json
{"exists":true,"selector":"#login-form"}
```

When the element is not found:

```json
{"exists":false,"selector":"#login-form"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Check if a login form exists:

```
hubcap exists '#login-form'
```

Conditionally click a dismiss button if it exists:

```
hubcap exists '.cookie-banner' | jq -r '.exists' | grep -q true && hubcap click '.cookie-banner .dismiss'
```

## See also

- [visible](visible.md) - Check if an element is visible
- [count](count.md) - Count matching elements
- [wait](wait.md) - Wait for an element to appear
- [query](query.md) - Query a DOM element
