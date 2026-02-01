# hubcap waitidle

Wait for network idle (no requests for the idle duration).

## When to use

Use `waitidle` to block until no network requests have been made for a specified duration. Use after page load to wait for async resources to finish. Use `waitload` for the load event specifically.

## Usage

```
hubcap waitidle [--idle <duration>]
```

## Arguments

None.

## Flags

| Flag   | Type     | Default | Description                                          |
|--------|----------|---------|------------------------------------------------------|
| --idle | duration | 500ms   | Time with no network activity to consider idle       |

## Output

| Field | Type | Description                           |
|-------|------|---------------------------------------|
| idle  | bool | Whether the network reached idle      |

```json
{"idle":true}
```

## Errors

| Condition                        | Exit code | Stderr                                    |
|----------------------------------|-----------|-------------------------------------------|
| Chrome not connected             | 2         | `error: connecting to Chrome: ...`        |
| Network not idle within timeout  | 3         | `error: timeout`                          |

## Examples

Wait for network idle with default 500ms threshold:

```
hubcap waitidle
```

Wait for a longer period of inactivity:

```
hubcap waitidle --idle 2s
```

Navigate and wait for all async resources to load (chaining):

```
hubcap goto https://example.com && hubcap waitidle --idle 1s
```

## See also

- [waitload](waitload.md) - Wait for the page load event
- [waitnav](waitnav.md) - Wait for any navigation event
- [network](network.md) - Monitor network activity
- [goto](goto.md) - Navigate to a URL
