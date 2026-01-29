# hubcap click

Click an element matching a CSS selector.

## When to use

Use `click` to activate buttons, links, and interactive elements by CSS selector. Prefer `clickat` when you have coordinates instead of a selector. Use `dblclick` for double-click interactions. For mobile touch simulation, use `tap` instead.

## Usage

```
hubcap click <selector>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the element to click |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `clicked` | boolean | Whether the click succeeded |
| `selector` | string | The selector that was clicked |

```json
{"clicked":true,"selector":"#btn"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Click a button by ID:

```
hubcap click '#submit'
```

Click a link by class:

```
hubcap click '.nav-link'
```

Click using an attribute selector:

```
hubcap click '[data-testid="login"]'
```

Wait for an element then click it:

```
hubcap wait '#modal-ok' && hubcap click '#modal-ok'
```

## See also

- [dblclick](dblclick.md) - Double-click an element
- [rightclick](rightclick.md) - Right-click an element
- [tripleclick](tripleclick.md) - Triple-click to select a paragraph
- [clickat](clickat.md) - Click at specific x,y coordinates
- [tap](tap.md) - Touch tap for mobile emulation
- [hover](hover.md) - Hover without clicking
