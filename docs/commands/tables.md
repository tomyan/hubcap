# hubcap tables

Extract structured table data from the current page.

## When to use

Use `tables` to get the headers and row data for every `<table>` element on the page. Returns structured arrays that are easy to process with `jq` or pipe into other tools.

## Usage

```
hubcap tables
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| tables | array | Array of table objects |
| tables[].id | string | The table's `id` attribute (omitted if none) |
| tables[].headers | array | Array of header cell strings |
| tables[].rows | array | Array of row arrays, each containing cell strings |

```json
{
  "tables": [
    {
      "id": "results",
      "headers": ["Name", "Score"],
      "rows": [
        ["Alice", "95"],
        ["Bob", "87"]
      ]
    }
  ]
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Extract all tables:

```
hubcap tables
```

Get the first table's data as CSV:

```
hubcap tables | jq -r '.tables[0] | (.headers | @csv), (.rows[] | @csv)'
```

Navigate to a page and extract a specific table by ID:

```
hubcap goto "https://example.com/data" && hubcap tables | jq '.tables[] | select(.id == "results")'
```

## See also

- [forms](forms.md) - list all forms and their input fields
- [links](links.md) - extract all links from the page
