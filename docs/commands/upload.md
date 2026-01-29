# hubcap upload

Upload files to a file input element.

## When to use

Use `upload` to set files on a `<input type="file">` element. Provide the file input selector and one or more local file paths. The command simulates the browser file-selection dialog without user interaction.

## Usage

```
hubcap upload <selector> <file>...
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `selector` | string | Yes | CSS selector of the file input element |
| `file` | string | Yes | One or more file paths to upload |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `uploaded` | boolean | Whether the upload succeeded |
| `selector` | string | The selector of the file input |
| `files` | array | List of file paths that were uploaded |

```json
{"uploaded":true,"selector":"#avatar","files":["photo.png"]}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Element not found | 1 | `error: no element found for selector: <sel>` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Upload a single file:

```
hubcap upload '#avatar' photo.png
```

Upload multiple files:

```
hubcap upload '#attachments' doc.pdf image.jpg
```

Upload using a full path:

```
hubcap upload '[type="file"]' /tmp/report.csv
```

Upload a file then click the submit button:

```
hubcap upload '#resume' ./resume.pdf && hubcap click '#apply'
```

## See also

- [fill](fill.md) - Fill a text input
- [click](click.md) - Click an element
- [forms](forms.md) - Get all form elements on the page
