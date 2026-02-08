# hubcap record

Record browser interactions as hubcap commands. Currently captures top-level page navigations.

## When to use

Use `record` to capture a browsing session and replay it later with `pipe`. Start recording, interact with the browser manually, then stop with Ctrl+C.

## Usage

```
hubcap record [--output <file>] [--duration <duration>]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --output | string | stdout | Write commands to file instead of stdout |
| --duration | duration | 0 (indefinite) | Recording duration; 0 means until Ctrl+C |

## Output

Outputs hubcap commands in pipe-compatible format:

```
# hubcap recording 2025-01-15T10:30:00Z
goto https://example.com
goto https://example.com/about
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Cannot create output file | 1 | `error: ...` |

## Examples

Record to stdout:

```
hubcap record
```

Record to a file:

```
hubcap record --output session.txt
```

Record for 30 seconds:

```
hubcap record --duration 30s --output session.txt
```

Replay a recorded session:

```
hubcap pipe < session.txt
```

## See also

- [pipe](pipe.md) - Replay recorded commands
- [shell](shell.md) - Interactive command entry
