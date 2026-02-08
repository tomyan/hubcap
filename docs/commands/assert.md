# hubcap assert

Assert page state. Exits 0 if the assertion passes, 1 with a descriptive error if it fails.

## When to use

Use `assert` in scripts and CI pipelines to verify expected page state. Combine with `retry` for flaky checks.

## Usage

```
hubcap assert <subcommand> <args...>
```

## Subcommands

| Subcommand | Usage | Description |
|------------|-------|-------------|
| text | `assert text <selector> <expected>` | Assert element text equals expected |
| title | `assert title <expected>` | Assert page title equals expected |
| url | `assert url <substring>` | Assert URL contains substring |
| exists | `assert exists <selector>` | Assert element exists in DOM |
| visible | `assert visible <selector>` | Assert element is visible |
| count | `assert count <selector> <n>` | Assert element count equals n |

## Output

| Field | Type | Description |
|-------|------|-------------|
| passed | bool | Always true on success |
| assertion | string | Description of what was asserted |

```json
{"passed":true,"assertion":"title == \"My Page\""}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Missing subcommand | 1 | `usage: hubcap assert <assertion> [args...]` |
| Unknown subcommand | 1 | `unknown assertion: <name>` |
| Assertion failed | 1 | Descriptive mismatch message |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |

## Examples

Assert the page title:

```
hubcap assert title "Welcome"
```

Assert an element exists:

```
hubcap assert exists '#login-form'
```

Assert element text:

```
hubcap assert text '#status' "Success"
```

Assert element count:

```
hubcap assert count '.item' 5
```

Assert URL contains a string:

```
hubcap assert url '/dashboard'
```

Combine with retry for eventual consistency:

```
hubcap retry --attempts 5 --interval 1s assert text '#status' "Done"
```

## See also

- [exists](exists.md) - Check element existence without asserting
- [visible](visible.md) - Check element visibility without asserting
- [retry](retry.md) - Retry a failing command
