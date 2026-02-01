# hubcap wait

Wait for an element matching a CSS selector to appear in the DOM.

## When to use

Use `wait` to block until an element matching a CSS selector exists in the DOM. Use `waittext` for text content. Use `waitgone` for element removal. Use `waitfn` for custom JavaScript conditions.

## Usage

```
hubcap wait <selector> [--timeout <duration>]
```

## Arguments

| Argument | Type   | Required | Description                          |
|----------|--------|----------|--------------------------------------|
| selector | string | Yes      | CSS selector of the element to wait for |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field    | Type   | Description                    |
|----------|--------|--------------------------------|
| found    | bool   | Whether the element was found  |
| selector | string | The selector that was matched  |

```json
{"found":true,"selector":".modal"}
```

## Errors

| Condition                          | Exit code | Stderr                                  |
|------------------------------------|-----------|------------------------------------------|
| Missing selector argument          | 1         | `usage: hubcap wait <selector> [--timeout <duration>]` |
| Chrome not connected               | 2         | `error: connecting to Chrome: ...`       |
| Element not found within timeout   | 3         | `error: timeout`                         |

## Examples

Wait for a modal to appear:

```
hubcap wait '.modal'
```

Wait with a custom timeout:

```
hubcap wait '#results' --timeout 10s
```

Click a button and wait for a result element (chaining):

```
hubcap click '#submit' && hubcap wait '.result' --timeout 15s
```

Wait for an element then extract its text:

```
hubcap wait '.notification' && hubcap text '.notification'
```

## See also

- [waittext](waittext.md) - Wait for text content to appear on the page
- [waitgone](waitgone.md) - Wait for an element to be removed from the DOM
- [waitfn](waitfn.md) - Wait for a JavaScript expression to return truthy
- [waitload](waitload.md) - Wait for the page load event
- [exists](exists.md) - Check if an element exists without waiting
