# hubcap shell

Interactive REPL for running hubcap commands. Displays a `hubcap>` prompt and executes commands as you type them.

## When to use

Use `shell` for exploratory browser automation — navigating pages, inspecting elements, and trying commands interactively without retyping `hubcap` each time.

## Usage

```
hubcap shell
```

## Dot commands

The shell supports special dot commands that aren't regular hubcap commands:

| Command | Description |
|---------|-------------|
| `.quit` | Exit the shell |
| `.exit` | Exit the shell (alias) |
| `.target <id>` | Switch target page by index or ID |
| `.output <format>` | Change output format (json, text, ndjson) |

## Examples

Start an interactive session:

```
$ hubcap shell
hubcap> goto https://example.com
hubcap> title
{"title":"Example Domain"}
hubcap> .output text
output set to "text"
hubcap> title
Example Domain
hubcap> .quit
```

Switch between tabs:

```
hubcap> tabs
hubcap> .target 1
target set to "1"
hubcap> title
```

## See also

- [pipe](pipe.md) - Non-interactive command input from stdin
