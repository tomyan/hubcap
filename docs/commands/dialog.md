# hubcap dialog

Handle browser dialogs (alert, confirm, prompt).

## When to use

Use `dialog` to accept or dismiss a currently active browser dialog. Must be called while a dialog is active. Use `--text` to provide input for prompt dialogs.

## Usage

```
hubcap dialog <action> [--text <prompt-text>]
```

## Arguments

| Argument | Type   | Required | Description                         |
|----------|--------|----------|-------------------------------------|
| action   | string | Yes      | Action to take: "accept" or "dismiss" |

## Flags

| Flag   | Type   | Default | Description                          |
|--------|--------|---------|--------------------------------------|
| --text | string | ""      | Text to enter for prompt dialogs     |

## Output

| Field      | Type   | Description                    |
|------------|--------|--------------------------------|
| action     | string | The action that was taken      |
| promptText | string | The text entered (if any)      |

```json
{"action":"accept"}
```

With prompt text:

```json
{"action":"accept","promptText":"my response"}
```

## Errors

| Condition            | Exit code | Stderr                         |
|----------------------|-----------|--------------------------------|
| Missing action       | 1         | `usage: hubcap dialog [accept\|dismiss] [--text <prompt-text>]` |
| Invalid action       | 1         | `action must be 'accept' or 'dismiss'` |
| Chrome not connected | 2         | `error: connecting to Chrome: ...`  |
| Timeout              | 3         | `error: timeout`               |

## Examples

Accept an alert:

```
hubcap dialog accept
```

Dismiss a confirm dialog:

```
hubcap dialog dismiss
```

Accept a prompt with input text:

```
hubcap dialog accept --text "my response"
```

Trigger a dialog via eval and accept it (chaining):

```
hubcap eval 'window.confirm("Are you sure?")' & hubcap dialog accept
```

## See also

- [eval](eval.md) - Evaluate JavaScript expressions
- [click](click.md) - Click an element
