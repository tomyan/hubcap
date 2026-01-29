# hubcap evalframe

Evaluate JavaScript in a specific frame.

## When to use

Use `evalframe` to run a JavaScript expression inside a particular iframe or frame context. Use `frames` to list available frame IDs first. Use `eval` when you need to evaluate JavaScript in the main frame only.

## Usage

```
hubcap evalframe <frame-id> <expression>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `frame-id` | string | Yes | Target frame identifier (use `frames` to list available IDs) |
| `expression` | string | Yes | JavaScript expression to evaluate |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `frameId` | string | The frame in which the expression was evaluated |
| `type` | string | JavaScript type of the result (e.g. `string`, `number`, `boolean`, `object`) |
| `value` | any | The returned value |

```json
{"frameId":"frame-123","type":"number","value":42}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Evaluate in a specific frame:

```
hubcap evalframe 'frame-123' 'document.title'
```

Get the body text of a frame:

```
hubcap evalframe 'frame-123' 'document.body.innerText'
```

Check the URL loaded in a frame:

```
hubcap evalframe 'frame-123' 'window.location.href'
```

List frames then evaluate in the first one:

```
hubcap frames | jq -r '.[0].id' | xargs -I{} hubcap evalframe '{}' 'document.title'
```

## See also

- [eval](eval.md) - Evaluate JavaScript in the main page context
- [frames](frames.md) - List all frames on the page
- [run](run.md) - Execute JavaScript from a file
