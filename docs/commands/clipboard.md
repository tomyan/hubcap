# hubcap clipboard

Read or write the system clipboard.

## When to use

Use `clipboard` to read from or write to the system clipboard. At least one of `--read` or `--write` is required.

## Usage

```
hubcap clipboard --read | --write <text>
```

## Arguments

None.

## Flags

| Flag    | Type   | Default | Description                    |
|---------|--------|---------|--------------------------------|
| --read  | bool   | false   | Read from clipboard            |
| --write | string | ""      | Text to write to clipboard     |

## Output

**Read mode** (`--read`):

| Field | Type   | Description              |
|-------|--------|--------------------------|
| text  | string | The clipboard contents   |

```json
{"text":"Hello, world!"}
```

**Write mode** (`--write`):

| Field   | Type   | Description            |
|---------|--------|------------------------|
| written | bool   | Whether the write succeeded |
| text    | string | The text that was written |

```json
{"written":true,"text":"Hello, world!"}
```

## Errors

| Condition                    | Exit code | Stderr                                  |
|------------------------------|-----------|------------------------------------------|
| No --read or --write given   | 1         | `error: --read or --write required`      |
| Chrome not connected         | 2         | `error: chrome not connected`            |
| Operation timeout            | 3         | `error: timeout`                         |

## Examples

Copy text to the clipboard:

```
hubcap clipboard --write "Hello, world!"
```

Read the clipboard:

```
hubcap clipboard --read
```

Extract clipboard text with jq (chaining):

```
hubcap clipboard --read | jq -r '.text'
```

Copy element text to clipboard:

```
hubcap text '#heading' | jq -r '.text' | xargs -I{} hubcap clipboard --write "{}"
```

## See also

- [cookies](cookies.md) - Manage browser cookies
- [storage](storage.md) - Get, set, or clear localStorage entries
- [session](session.md) - Get, set, or clear sessionStorage entries
