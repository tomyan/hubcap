# hubcap pdf

Export the current page as a PDF document.

## When to use

Use `pdf` to export the current page as a PDF file. Use `screenshot` for image capture instead.

## Usage

```
hubcap pdf --output <file> [flags]
```

## Arguments

None.

## Flags

| Flag         | Type   | Default | Description                |
|--------------|--------|---------|----------------------------|
| --output     | string | ""      | File path to save the PDF (required) |
| --landscape  | bool   | false   | Landscape orientation      |
| --background | bool   | false   | Print background graphics  |

## Output

| Field     | Type   | Description                    |
|-----------|--------|--------------------------------|
| output    | string | Path to the saved PDF file     |
| size      | int    | File size in bytes             |
| landscape | bool   | Whether landscape was used     |

```json
{"output":"page.pdf","size":54321,"landscape":false}
```

## Errors

| Condition            | Exit code | Stderr                          |
|----------------------|-----------|---------------------------------|
| Missing --output     | 1         | `error: --output flag required` |
| Chrome not connected | 2         | `error: chrome not connected`   |
| Export timeout       | 3         | `error: pdf export timeout`     |

## Examples

Export a basic PDF:

```
hubcap pdf --output page.pdf
```

Export in landscape with backgrounds:

```
hubcap pdf --output report.pdf --landscape --background
```

Navigate to a page and export as PDF (chaining):

```
hubcap goto --wait https://example.com && hubcap pdf --output example.pdf
```

## See also

- [screenshot](screenshot.md) - Capture a screenshot of the page
- [goto](goto.md) - Navigate to a URL
