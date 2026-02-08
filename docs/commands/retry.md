# hubcap retry

Retry a command on failure. Runs the inner command up to N times with a configurable interval between attempts.

## When to use

Use `retry` to handle transient failures — elements that appear after a delay, APIs that occasionally time out, or pages that load slowly. Combine with `assert` for robust checks.

## Usage

```
hubcap retry [--attempts N] [--interval duration] <command> [args...]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --attempts | int | 3 | Maximum number of attempts |
| --interval | duration | 1s | Delay between attempts |

## Output

On success, outputs the inner command's result. On failure after all attempts, exits with the inner command's last exit code.

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Missing command | 1 | `usage: hubcap retry [--attempts N] [--interval duration] <command> [args...]` |
| Unknown command | 1 | `unknown command: <name>` |
| All attempts fail | varies | Inner command's error output |

## Examples

Retry waiting for an element (3 attempts, 1s apart):

```
hubcap retry wait '#results'
```

Retry with more attempts and longer interval:

```
hubcap retry --attempts 10 --interval 2s assert exists '.loaded'
```

Retry a title assertion:

```
hubcap retry --attempts 5 assert title "Dashboard"
```

## See also

- [assert](assert.md) - Assert page state
- [wait](wait.md) - Wait for an element to appear
