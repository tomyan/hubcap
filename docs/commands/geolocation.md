# hubcap geolocation

Override the browser's geolocation.

## When to use

Override the browser's geolocation. Useful for testing location-dependent features. Set `permission geolocation granted` first if the page requests geolocation permission.

## Usage

```
hubcap geolocation <latitude> <longitude>
```

## Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `latitude` | float | Yes | Latitude in decimal degrees (-90 to 90) |
| `longitude` | float | Yes | Longitude in decimal degrees (-180 to 180) |

## Flags

None.

## Output

| Field | Type | Description |
|-------|------|-------------|
| `latitude` | number | The latitude that was set |
| `longitude` | number | The longitude that was set |
| `accuracy` | number | Geolocation accuracy in meters (always 1) |

```json
{"latitude":37.7749,"longitude":-122.4194,"accuracy":1}
```

## Errors

| Condition | Exit code | Stderr |
|-----------|-----------|--------|
| Invalid coordinate (non-numeric or out of range) | 1 | `error: invalid coordinate` |
| Chrome not connected | 2 | `error: connecting to Chrome: ...` |
| Timeout | 3 | `error: timeout` |

## Examples

Set location to San Francisco:

```
hubcap geolocation 37.7749 -122.4194
```

Set location to Tokyo:

```
hubcap geolocation 35.6762 139.6503
```

Grant geolocation permission, set coordinates, then reload to apply:

```
hubcap permission geolocation granted && hubcap geolocation 51.5074 -0.1278 && hubcap reload
```

## See also

- [permission](permission.md) - Grant or deny browser permissions
- [emulate](emulate.md) - Emulate a full device profile
- [viewport](viewport.md) - Set the browser viewport size
