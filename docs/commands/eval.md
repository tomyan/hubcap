# hubcap eval

Evaluate a JavaScript expression in the page context and return the result.

## When to use

Evaluate a JavaScript expression in the page context. Returns the result value and type. Use `run` to execute a JS file instead of an inline expression. Use `evalframe` to evaluate inside a specific frame.

## Usage

```
hubcap eval <expression>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `expression` | string | Yes | JavaScript expression to evaluate in the page context |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `value` | any | The returned value from the expression |
| `type` | string | JavaScript type of the result (`string`, `number`, `boolean`, `object`, `undefined`) |

```json
{"value":42,"type":"number"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| JavaScript evaluation error | 1 | `eval: <error message>` |
| Chrome not connected | 2 | `chrome: not connected` |
| Operation timed out | 3 | `timeout` |

## Examples

Get the page title:

```
hubcap eval 'document.title'
```

Count elements matching a selector:

```
hubcap eval 'document.querySelectorAll("a").length'
```

Return a JSON object:

```
hubcap eval '({width: window.innerWidth, height: window.innerHeight})'
```

Chain with jq to extract the value and use in a shell pipeline:

```
hubcap eval 'document.querySelectorAll("li").length' | jq -r '.value'
```

## See also

- [run](run.md) - Execute JavaScript from a file
- [evalframe](evalframe.md) - Evaluate JavaScript in a specific frame
- [waitfn](waitfn.md) - Wait for a JavaScript function to return truthy
