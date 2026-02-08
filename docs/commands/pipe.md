# hubcap pipe

Read commands from stdin, one per line. Each command opens its own connection to Chrome.

## When to use

Use `pipe` to run a sequence of hubcap commands from a file or script without repeating the `hubcap` prefix. Blank lines and lines starting with `#` are skipped.

## Usage

```
hubcap pipe
```

## Input format

One command per line. Arguments support double and single quotes for values containing spaces.

```
# Navigate and interact
goto https://example.com
wait '#content'
fill '#search' "hello world"
click '#submit'
title
```

## Output

Each command's JSON output is written to stdout in sequence.

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Unknown command in input | continues | `unknown command: <name>` |
| Command fails | varies | Stops and exits with failing command's code |
| Stdin read error | 1 | `error reading stdin: ...` |

## Examples

Run commands from a file:

```
hubcap pipe < commands.txt
```

Pipe from a heredoc:

```
hubcap pipe <<'EOF'
goto https://example.com
wait '#main'
title
screenshot --output page.png
EOF
```

Generate commands dynamically:

```
echo "goto https://example.com" | hubcap pipe
```

## See also

- [shell](shell.md) - Interactive REPL with prompt
- [record](record.md) - Record browser interactions as pipe-compatible commands
