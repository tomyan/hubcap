# hubcap waitgone

Wait for an element to be removed from the DOM.

## When to use

Use `waitgone` to block until an element matching a CSS selector is no longer present in the DOM. Use after dismissing dialogs or closing modals. Use `wait` to wait for an element to appear instead.

## Usage

```
hubcap waitgone <selector> [--timeout <duration>]
```

## Arguments

| Argument | Type   | Required | Description                                 |
|----------|--------|----------|---------------------------------------------|
| selector | string | Yes      | CSS selector of the element to wait for removal |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field    | Type   | Description                       |
|----------|--------|-----------------------------------|
| gone     | bool   | Whether the element was removed   |
| selector | string | The selector that was waited on   |

```json
{"gone":true,"selector":".spinner"}
```

## Errors

| Condition                            | Exit code | Stderr                                    |
|--------------------------------------|-----------|-------------------------------------------|
| Missing selector argument            | 1         | `usage: hubcap waitgone <selector> [--timeout <duration>]` |
| Chrome not connected                 | 2         | `error: connecting to Chrome: ...`        |
| Element still present after timeout  | 3         | `error: timeout`                          |

## Examples

Wait for a loading spinner to disappear:

```
hubcap waitgone '.spinner'
```

Wait for an overlay to be removed:

```
hubcap waitgone '#overlay' --timeout 15s
```

Dismiss a modal and wait for it to be removed (chaining):

```
hubcap click '.modal-close' && hubcap waitgone '.modal'
```

## See also

- [wait](wait.md) - Wait for an element to appear in the DOM
- [waittext](waittext.md) - Wait for text content to appear on the page
- [waitfn](waitfn.md) - Wait for a JavaScript expression to return truthy
- [exists](exists.md) - Check if an element exists without waiting
