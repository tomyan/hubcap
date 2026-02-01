# hubcap waittext

Wait for specific text to appear anywhere on the page.

## When to use

Use `waittext` to block until a given text string appears in the page content. Use `wait` for element selectors. Use `find` to search without waiting.

## Usage

```
hubcap waittext <text> [--timeout <duration>]
```

## Arguments

| Argument | Type   | Required | Description                  |
|----------|--------|----------|------------------------------|
| text     | string | Yes      | The text string to wait for  |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field | Type   | Description                   |
|-------|--------|-------------------------------|
| text  | string | The text that was matched     |
| found | bool   | Whether the text was found    |

```json
{"text":"Order confirmed","found":true}
```

## Errors

| Condition                      | Exit code | Stderr                                |
|--------------------------------|-----------|---------------------------------------|
| Missing text argument          | 1         | `usage: hubcap waittext <text> [--timeout <duration>]` |
| Chrome not connected           | 2         | `error: connecting to Chrome: ...`    |
| Text not found within timeout  | 3         | `error: timeout`                      |

## Examples

Wait for a success message:

```
hubcap waittext 'Order confirmed'
```

Wait with a short timeout:

```
hubcap waittext 'Loading complete' --timeout 5s
```

Submit a form and wait for confirmation (chaining):

```
hubcap click '#submit' && hubcap waittext 'Thank you for your order'
```

## See also

- [wait](wait.md) - Wait for an element by CSS selector
- [waitgone](waitgone.md) - Wait for an element to be removed from the DOM
- [find](find.md) - Search for text without waiting
- [text](text.md) - Get text content of an element
