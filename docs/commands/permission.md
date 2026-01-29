# hubcap permission

Set a browser permission to granted, denied, or prompt.

## When to use

Set browser permission state for features that require user consent. Available permissions: geolocation, notifications, camera, microphone, midi, push. Set the permission before navigating to or interacting with pages that request it, so the browser does not show a permission prompt.

## Usage

```
hubcap permission <name> <state>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Permission name: `geolocation`, `notifications`, `camera`, `microphone`, `midi`, or `push` |
| `state` | string | Yes | Permission state: `granted`, `denied`, or `prompt` |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `permission` | string | The permission that was configured |
| `state` | string | The state it was set to |

```json
{"permission":"geolocation","state":"granted"}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Invalid state value | 1 | `error: state must be "granted", "denied", or "prompt"` |
| Chrome not connected | 2 | `error: chrome connection failed` |
| Timeout waiting for response | 3 | `error: timeout` |

## Examples

Grant geolocation access:

```
hubcap permission geolocation granted
```

Deny notification prompts:

```
hubcap permission notifications denied
```

Reset camera to prompt:

```
hubcap permission camera prompt
```

Grant geolocation, set coordinates, then verify the page reads the location:

```
hubcap permission geolocation granted && hubcap geolocation 37.7749 -122.4194 && hubcap eval "navigator.geolocation.getCurrentPosition(p => document.title = p.coords.latitude)"
```

## See also

- [geolocation](geolocation.md) - Set geolocation coordinates (pair with `permission geolocation granted`)
- [emulate](emulate.md) - Emulate a full device profile
