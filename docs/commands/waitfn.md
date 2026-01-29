# hubcap waitfn

Wait for a JavaScript expression to return a truthy value.

## When to use

Use `waitfn` to block until a JavaScript expression evaluates to a truthy value. Use for complex conditions not covered by `wait` or `waittext`.

## Usage

```
hubcap waitfn <expression> [--timeout <duration>]
```

## Arguments

| Argument   | Type   | Required | Description                                      |
|------------|--------|----------|--------------------------------------------------|
| expression | string | Yes      | JavaScript expression to evaluate until truthy   |

## Flags

| Flag      | Type     | Default | Description         |
|-----------|----------|---------|---------------------|
| --timeout | duration | 30s     | Maximum wait time   |

## Output

| Field      | Type   | Description                            |
|------------|--------|----------------------------------------|
| completed  | bool   | Whether the expression returned truthy |
| expression | string | The expression that was evaluated      |

```json
{"completed":true,"expression":"window.appReady === true"}
```

## Errors

| Condition                              | Exit code | Stderr                                    |
|----------------------------------------|-----------|-------------------------------------------|
| Expression not truthy within timeout   | 3         | `error: timeout waiting for expression`   |
| Chrome not connected                   | 2         | `error: chrome not connected`             |
| Missing expression argument            | 1         | `error: expression argument required`     |

## Examples

Wait for an app-level ready flag:

```
hubcap waitfn 'window.appReady === true'
```

Wait for a specific array length:

```
hubcap waitfn 'document.querySelectorAll(".item").length >= 10' --timeout 20s
```

Navigate and wait for a custom condition (chaining):

```
hubcap goto --wait https://example.com && hubcap waitfn 'window.dataLoaded'
```

## See also

- [wait](wait.md) - Wait for an element by CSS selector
- [waittext](waittext.md) - Wait for text content to appear on the page
- [eval](eval.md) - Evaluate a JavaScript expression
- [waitgone](waitgone.md) - Wait for an element to be removed from the DOM
