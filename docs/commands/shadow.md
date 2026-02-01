# hubcap shadow

Query an element inside a shadow DOM.

## When to use

Use `shadow` to locate an element inside a shadow root by providing the host element selector and then the inner selector. This is required for web components that encapsulate their DOM. Use `query` for elements in the regular (light) DOM.

## Usage

```
hubcap shadow <host-selector> <inner-selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `host-selector` | string | Yes | CSS selector for the shadow DOM host element |
| `inner-selector` | string | Yes | CSS selector to match inside the shadow root |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `hostSelector` | string | The host element selector that was used |
| `innerSelector` | string | The inner selector that was matched |
| `nodeId` | number | Internal node identifier |
| `tagName` | string | Element tag name in uppercase |
| `attributes` | object | Key-value map of the element's HTML attributes |

```json
{"hostSelector":"my-component","innerSelector":"button.inner","nodeId":123,"tagName":"BUTTON","attributes":{"class":"inner"}}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Host element not found | 1 | `error: shadow host not found: <sel>` |
| No shadow root on host | 1 | `error: no shadow root found on element: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Query a button inside a custom element's shadow root:

```
hubcap shadow 'my-component' 'button.inner'
```

Get an input inside a shadow host identified by ID:

```
hubcap shadow '#player' 'input[type=range]'
```

Query a shadow DOM element then extract its tag name:

```
hubcap shadow 'my-component' '.label' | jq -r '.tagName'
```

## See also

- [query](query.md) - Query elements in the regular DOM
- [html](html.md) - Get outer HTML of an element
- [text](text.md) - Get the inner text of an element
