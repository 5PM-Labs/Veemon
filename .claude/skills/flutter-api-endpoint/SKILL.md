---
name: flutter-api-endpoint
description: Call a Go API route (declared in contract/*.proto) from the Flutter app apps/mobile — DTO matching the protojson wire shape, ApiEndpoints path, remote data source method, repository guard() mapping, use case, and tests. Use when apps/mobile needs a new or changed backend call, or after a proto contract change that mobile consumes.
paths:
  - "apps/mobile/**"
  - "contract/**"
---

# Wire a contract route into `apps/mobile`

The proto contract (`contract/<svc>/<svc>.proto`) is the source of truth.
Mobile has no generated client: the DTOs and paths are written by hand and
**must match the contract**. If the route doesn't exist yet, add it with the
`proto-routes` skill first.

## 1. Read the contract

For the RPC, note:

- `veemon.route` → `method`, `path` (with `{param}`s), `auth { required, roles }`,
  `body`, and `response` style.
- Request and response messages → field names **as `json_name`** (camelCase)
  and types.

The server marshals with `protojson.MarshalOptions{EmitUnpopulated: true}`
(`internal/pkg/response/response.go`): empty strings and zeros **are** sent,
and unset nested messages arrive as `null`.

The Go server wraps every response in the `internal/pkg/response` envelope
`{success, data, meta?}` / `{success:false, error:{code,message}}`.
`ApiClient` unwraps it and hands you `data`.

## 2. Endpoint path

```dart
// lib/core/network/api_endpoints.dart
static const String orders = '/api/v1/orders';
static String order(String id) => '/api/v1/orders/${Uri.encodeComponent(id)}';
```

Always encode path parameters. Never put tokens or PII in query strings.

## 3. DTO = the proto message's wire shape

```dart
// lib/features/<f>/data/dtos/order_dto.dart
@freezed
abstract class OrderDto with _$OrderDto {
  const factory OrderDto({
    required String id,
    @Default('') String note,        // defaults keep parsing tolerant
    @Default(0) int quantity,
    String? createdAt,               // timestamps arrive as strings; parse in the mapper
  }) = _OrderDto;

  factory OrderDto.fromJson(Map<String, dynamic> json) => _$OrderDtoFromJson(json);
}
```

| proto | Dart DTO field |
|-------|----------------|
| `string` | `@Default('') String` (or `required String` for IDs) |
| `int32` / `uint32` | `@Default(0) int` |
| `int64` / `uint64` | **`String`** in protojson → parse in the mapper (`int.tryParse`) |
| `bool` | `@Default(false) bool` |
| `double` / `float` | `@Default(0) double` |
| `repeated T` | `@Default(<TDto>[]) List<TDto>` |
| nested message | `TDto?` (`null` when unset) |
| `enum` | `String` → map to a Dart enum in the mapper, with an `unknown` fallback |
| `google.protobuf.Timestamp` / RFC3339 string | `String?` → `DateTime.tryParse` in the mapper |

Note: protojson encodes 64-bit ints as JSON strings. Typing them as `int`
fails parsing. `Pagination.total` (`int64`) is one of these.

Add a `toEntity()` extension in `data/mappers/`. Entities never see wire
quirks.

## 4. Data source method

```dart
@override
Future<List<OrderDto>> getOrders({required int page, required int size}) async {
  final data = await _client.get('${ApiEndpoints.orders}?page=$page&size=$size');
  return _parse(() => [
    for (final json in (data['orders'] as List<dynamic>? ?? const []))
      OrderDto.fromJson(json as Map<String, dynamic>),
  ]);
}
```

- `ApiClient.get` / `ApiClient.post` only. If you need PUT/DELETE, add
  `put`/`delete` to `ApiClient` with the same `_send` wrapper (plus a test in
  `test/core/network/api_client_test.dart`).
- Wrap parsing in `_parse` so a contract mismatch becomes
  `MalformedResponseException` → `Failure.unexpected()`.
- Auth routes need nothing extra. `AuthInterceptor` attaches the bearer token.

## 5. Repository mapping

```dart
@override
Future<Either<Failure, List<OrderEntity>>> getOrders({int page = 1}) => guard(
  () async => [for (final dto in await _remote.getOrders(page: page, size: 20)) dto.toEntity()],
  // Endpoint-specific statuses only; return null for the default mapping.
  onApiException: (e) => e.statusCode == 404 ? const Failure.server(statusCode: 404) : null,
);
```

Default mapping: 401 → `unauthorized`, 429 → `rateLimited`, others →
`server`. If the route is role-gated (`roles: [...]`), 403 maps to `server`
by default. Add a `forbidden` failure only if the UI must react to it
specifically.

## 6. Tests

- `data/repositories/*_impl_test.dart`: success mapping + each exception →
  failure (table-driven), and side effects.
- `domain/use_cases/*_test.dart`: rules and delegation.
- If you touched `ApiClient`: an envelope test with the fake adapter.
- Fixtures use synthetic data, with wire-accurate JSON shapes (camelCase,
  int64 as string).

## 7. Verify and keep in sync

```bash
make -C apps/mobile gen && make -C apps/mobile check
```

A proto change that renames or removes a field mobile reads is a **breaking
change** for shipped app versions (users don't update instantly). Add new
fields; deprecate before removing; version the path (`/api/v2`) for breaking
shape changes.
