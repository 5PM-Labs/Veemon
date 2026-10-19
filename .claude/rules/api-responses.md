# API Response Standards

REST payloads are serialized with `protojson` via `internal/pkg/response` helpers, so
`data` field names use **camelCase** (matching the proto JSON mapping).

## Success Response

```json
{
    "success": true,
    "data": { ... },
    "meta": {
        "page": 1,
        "size": 10,
        "total": 100
    }
}
```

## Error Response

```json
{
    "success": false,
    "error": {
        "code": 40001,
        "message": "validation failed: email is required"
    }
}
```
