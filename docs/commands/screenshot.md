# hubcap screenshot

Capture a screenshot of the page or a specific element.

## When to use

Use `screenshot` to capture an image of the current page or a specific element selected by CSS. Use `--base64` to get inline image data instead of writing to a file. Use `pdf` for PDF export instead.

## Usage

```
hubcap screenshot --output <file> [flags]
hubcap screenshot --base64 [flags]
```

## Arguments

None.

## Flags

| Flag       | Type   | Default | Description                              |
|------------|--------|---------|------------------------------------------|
| --output   | string | ""      | File path to save the screenshot (required unless --base64) |
| --format   | string | "png"   | Image format: png, jpeg, or webp         |
| --quality  | int    | 80      | JPEG/WebP quality 0-100                  |
| --selector | string | ""      | CSS selector for element screenshot      |
| --base64   | bool   | false   | Return base64 data instead of file       |

## Output

| Field  | Type   | Description                                    |
|--------|--------|------------------------------------------------|
| format | string | Image format used                              |
| size   | int    | File size in bytes                             |
| data   | string | Base64-encoded image data (only with --base64) |

Default output:

```json
{"format":"png","size":12345}
```

With `--base64`:

```json
{"format":"png","size":12345,"data":"iVBOR..."}
```

## Errors

| Condition              | Exit code | Stderr                                |
|------------------------|-----------|---------------------------------------|
| No output or base64    | 1         | `error: --output or --base64 required`|
| Selector not found     | 1         | `error: element not found: <sel>`     |
| Chrome not connected   | 2         | `error: connecting to Chrome: ...`    |
| Timeout                | 3         | `error: timeout`                      |

## Examples

Capture a full-page screenshot:

```
hubcap screenshot --output page.png
```

Capture as JPEG with reduced quality:

```
hubcap screenshot --output page.jpg --format jpeg --quality 50
```

Capture a specific element:

```
hubcap screenshot --output hero.png --selector '.hero-banner'
```

Capture and return base64 data:

```
hubcap screenshot --base64 --format webp
```

Navigate to a page and screenshot it (chaining):

```
hubcap goto --wait https://example.com && hubcap screenshot --output example.png
```

## See also

- [pdf](pdf.md) - Export the page as a PDF document
- [viewport](viewport.md) - Get or set the browser viewport size
- [emulate](emulate.md) - Emulate a device
