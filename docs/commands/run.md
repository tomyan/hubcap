# hubcap run

Execute JavaScript from a file in the page context.

## When to use

Execute JavaScript from a file in the page context. The file contents are read and evaluated as a single expression. Use `eval` for inline expressions instead of loading a file.

## Usage

```
hubcap run <file>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `file` | string | Yes | Path to a `.js` file to execute |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `file` | string | Path to the executed file |
| `value` | any | The returned value from the script |

```json
{"file":"script.js","value":42}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| File not found | 1 | `run: file not found: <path>` |
| JavaScript evaluation error | 1 | `run: <error message>` |
| Chrome not connected | 2 | `chrome: not connected` |
| Operation timed out | 3 | `timeout` |

## Examples

Run a script:

```
hubcap run setup.js
```

Run a script and extract the result with jq:

```
hubcap run check.js | jq '.value'
```

Use a script that returns page data, then pipe into further processing:

```
hubcap run scrape.js | jq -r '.value[]' | sort -u > links.txt
```

## See also

- [eval](eval.md) - Evaluate an inline JavaScript expression
- [evalframe](evalframe.md) - Evaluate JavaScript in a specific frame
