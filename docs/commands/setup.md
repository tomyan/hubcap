# hubcap setup

Configure profiles and manage Chrome connections.

## When to use

Use `setup` to create, manage, and switch between named connection profiles. Profiles store Chrome connection settings (host, port, headless mode, etc.) so you don't need to pass flags every time.

## Usage

```
hubcap setup                    Show current config (non-TTY) or run wizard (TTY)
hubcap setup list               List all profiles
hubcap setup show [name]        Show profile details (default: active profile)
hubcap setup add <name>         Add a new profile
hubcap setup edit <name>        Edit an existing profile
hubcap setup remove <name>      Remove a profile
hubcap setup default [name]     Get or set the default profile
hubcap setup status [name]      Check Chrome connectivity for a profile
hubcap setup launch [name]      Launch Chrome for a profile
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| subcommand | no | One of: list, show, add, edit, remove, default, status, launch |
| name | varies | Profile name (required for add, edit, remove; optional for others) |

## Flags (add/edit)

| Flag | Type | Description |
|------|------|-------------|
| --host | string | Chrome debug host |
| --port | int | Chrome debug port |
| --timeout | string | Command timeout (e.g. "30s") |
| --output | string | Output format |
| --chrome-path | string | Path to Chrome binary |
| --headless | bool | Run Chrome in headless mode |
| --chrome-data-dir | string | Chrome user data directory |
| --ephemeral | bool | Auto-launch and cleanup Chrome |
| --ephemeral-timeout | string | Idle timeout for ephemeral sessions (e.g. "10m") |
| --set-default | bool | Set this profile as default |

## Flags (remove)

| Flag | Type | Description |
|------|------|-------------|
| --force | bool | Skip confirmation prompt |

## Output

Output format depends on the subcommand. All subcommands support `--output json`.

### setup list

```json
[
  {"name": "local", "host": "localhost", "port": 9222, "is_default": true},
  {"name": "ci", "host": "ci-host", "port": 9333}
]
```

### setup show

```json
{
  "name": "local",
  "host": "localhost",
  "port": 9222,
  "is_default": true
}
```

### setup status

```json
{
  "profile": "local",
  "host": "localhost",
  "port": 9222,
  "connected": true
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Profile not found | 1 | `error: profile "name" not found` |
| Duplicate profile name | 1 | `error: profile "name" already exists` |
| Unknown subcommand | 1 | `unknown setup subcommand: ...` |

## Examples

Create a profile for local development:

```
hubcap setup add local --host localhost --port 9222 --set-default
```

Create a headless CI profile:

```
hubcap setup add ci --port 9333 --headless --ephemeral --ephemeral-timeout 10m
```

Check if Chrome is reachable for a profile:

```
hubcap setup status local
```

Switch the default profile:

```
hubcap setup default ci
```

Use a specific profile for a command:

```
hubcap --profile ci title
```

## Profile storage

Profiles are stored in `~/.config/hubcap/profiles.json`. Override the config directory with `$HUBCAP_CONFIG_DIR`.

## Config precedence

```
Built-in defaults < Named profile < .hubcaprc < Environment vars < CLI flags
```

## See also

- [help](help.md) - show help for a command
