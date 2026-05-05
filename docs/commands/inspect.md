# hubcap inspect

Open a terminal-based web inspector that streams console output, page state, and tab info from Chrome.

## When to use

Use `inspect` interactively to monitor a Chrome tab in real time: console messages and errors stream into the terminal, the page title and URL update as you navigate, and you can switch between tabs without leaving the TUI. Useful for ambient observation while debugging or driving the browser from another shell.

The inspector auto-reconnects if Chrome restarts or the target tab is closed.

## Usage

```
hubcap inspect
```

## Flags

| Flag | Description |
|------|-------------|
| --target | Target page (index or ID); inherited from the global flag |

## Examples

Open the inspector against the default tab:

```
hubcap inspect
```

Inspect a specific tab by ID:

```
hubcap --target ABC123 inspect
```

Quit the inspector with the standard `q` or `Ctrl+C` key.

## See also

- `hubcap console` — capture console messages without a TUI
- `hubcap tabs` — list open tabs to find a target ID
- `hubcap shell` — interactive REPL for issuing commands
