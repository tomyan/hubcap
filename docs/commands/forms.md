# hubcap forms

List all forms and their input fields on the current page.

## When to use

Use `forms` to understand the structure of forms on a page before filling them programmatically with `fill`, `select`, or `check`. Returns each form's action, method, and all of its input elements.

## Usage

```
hubcap forms
```

## Arguments

None.

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| forms | array | Array of form objects |
| forms[].id | string | The form's `id` attribute |
| forms[].action | string | The form's `action` URL |
| forms[].method | string | The form's `method` (GET, POST, etc.) |
| forms[].inputs | array | Array of input field objects |
| count | number | Total number of forms found |

```json
{
  "forms": [
    {
      "id": "login",
      "action": "/auth",
      "method": "POST",
      "inputs": [
        {"name": "username", "type": "text", "value": ""},
        {"name": "password", "type": "password", "value": ""},
        {"name": "remember", "type": "checkbox", "value": "on"}
      ]
    }
  ],
  "count": 1
}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

List all forms on the page:

```
hubcap forms
```

Discover the login form fields, then fill and submit:

```
hubcap forms | jq '.forms[] | select(.id == "login") | .inputs[].name'
hubcap fill "#username" "admin" && hubcap fill "#password" "secret" && hubcap click "button[type=submit]"
```

Count the total number of input fields across all forms:

```
hubcap forms | jq '[.forms[].inputs | length] | add'
```

## See also

- [fill](fill.md) - fill in a form field
- [select](select.md) - select an option in a dropdown
- [check](check.md) - check a checkbox
- [tables](tables.md) - extract structured table data
