# Veemon API

Go 1.26.9 backend: Fiber v3 for REST and gRPC, proto-first route generation, PASETO v4 tokens, fail-closed auth.

[← back to root README](../../README.md)

## Running it

Prerequisites:

| Tool | Required for |
|------|---------------|
| Go 1.26.9 | everything |
| Docker + `docker compose` | Postgres, Redis, RabbitMQ, Jaeger |
| `make` | the command reference below |
| `buf` + `protoc-gen-go` + `protoc-gen-go-grpc` | only when regenerating routes from `.proto` |

Start the infrastructure, then run the API from the `apps/api` directory:

```bash
docker compose up -d

cp apps/api/.env.example apps/api/.env    # from the repo root
cd apps/api
make deps
make migrate
make run
```

`buf generate` is invoked by the `proto` target and must run from the repo root; the target handles that itself:

```bash
make -C apps/api proto
```

There is no `.env.example` at the repo root — the template lives at `apps/api/.env.example`.

### Ports

| Port | Service |
|------|---------|
| `3000` | REST (HTTP) |
| `50051` | gRPC |

Health and admin endpoints are on the REST port: `/health`, `/ready`, `/metrics`, `/docs`.

### Configuration

Config is read by Viper (`config/config.go`). Every key is bound explicitly to its environment variable, so env-only overrides work. `PASETO_SECRET`, `PASETO_EXPIRATION_HOURS`, and the connection settings are all read from the process environment or the `.env` file.

Optional: load secrets from Infisical at startup (`config/infisical.go`), or run the process under the Infisical CLI with `make infisical-run` (`INFISICAL_ENV`, `INFISICAL_PATH` are Makefile variables).

Config validation runs at startup and **fails fast** — see [Security](#security).

## Architecture

Two transports, one set of handlers:

```
client ──HTTP──▶ Fiber v3 ──┐
                              ├─▶ handler ─▶ usecase ─▶ repository ─▶ entity ─▶ Postgres
client ──gRPC──▶ gRPC srv ───┘        │
                                       └─▶ entity (no repo for pure reads)
```

**Composition root.** `cmd/server` and `cmd/worker` are thin entry points — `fx.New(ServerModule).Run()`. All wiring lives in `config/fx_*.go`, and the concrete constructors are in `config/bootstrap.go` (`provideTokenService`, `provideAuthGuard`, `provideTokenValidator`, `provideGRPCServer`, `registerHTTPRoutes`, `registerObservabilityAndHealthRoutes`).

**Layering.** `handler` → `app/usecase` → `repository` → `entity`. Handlers do binding and envelope writing, usecases hold business rules, repositories own queries, entities are plain structs.

**Routes do not live in the handler directory.** Fiber routes are generated into `handler/grpc/<svc>/` from the proto contract, and mounted by `pb_user.RegisterUserApiRoutes(app, userHandler, tokenValidator)` in `config/bootstrap.go`. Adding a route means editing a `.proto`, never a route table.

The worker (`cmd/worker`) is a separate `fx` module that shares the same config, logging, database, Redis, and RabbitMQ infrastructure.

## Proto-first routes

The REST surface is declared on the RPC itself. `contract/veemon/annotations.proto` defines the `Route` option:

```protobuf
message Route {
  // HTTP verb: GET, POST, PUT, PATCH, or DELETE.
  string method = 1;

  // Path template, e.g. "/api/v1/users/{id}". {segments} become path params.
  string path = 2;

  // When true, the JSON request body is parsed into the request message
  // (typically for POST/PUT/PATCH). Ignored for bodyless requests.
  bool body = 3;

  // Authentication/authorization policy for this route. Absent means public.
  Auth auth = 4;

  // How the successful response is written. Defaults to a 200 OK envelope.
  ResponseStyle response = 5;

  // Optional per-route rate limit applied before the handler. Useful for
  // unauthenticated credential endpoints (login/register) to blunt brute force.
  RateLimit rate_limit = 6;
}

message Auth {
  // When true, a valid token is required to reach the handler.
  bool required = 1;

  // If non-empty, the caller must hold at least one of these roles. Implies
  // required = true.
  repeated string roles = 2;
}

enum ResponseStyle {
  // 200 OK, single message under `data`.
  RESPONSE_STYLE_OK = 0;

  // 201 Created, single message under `data`.
  RESPONSE_STYLE_CREATED = 1;

  // 200 OK list envelope: the response's single repeated field becomes `data`
  // and its pagination message (if any) becomes `meta`.
  RESPONSE_STYLE_LIST = 2;
}

message RateLimit {
  // Maximum requests allowed per window.
  uint32 max = 1;

  // Window length in seconds.
  uint32 window_seconds = 2;
}

extend google.protobuf.MethodOptions {
  // Field number in the internal (50000-99999) extension range.
  Route route = 50123;
}
```

Attached to an RPC, from `contract/user/user.proto`:

```protobuf
rpc Login(LoginReq) returns (LoginRes) {
    option (veemon.route) = {
        method: "POST"
        path: "/api/v1/auth/login"
        body: true
        rate_limit: { max: 10 window_seconds: 60 }
    };
}
```

Path parameters use `{name}` and are matched against request fields by `json_name`; the plugin converts them to Fiber's `:name` syntax. For `GET`, request fields not bound to a path parameter are read from the query string.

### What codegen produces

`make proto` builds `./cmd/protoc-gen-fiber` into `apps/api/bin`, then runs `buf generate` from the repo root (`buf.gen.yaml`). Outputs:

| Plugin | Output path |
|--------|-------------|
| `protoc-gen-go` | `apps/api/handler/grpc/user/user.pb.go` |
| `protoc-gen-go-grpc` | `apps/api/handler/grpc/user/user_grpc.pb.go` |
| `protoc-gen-fiber` | `apps/api/handler/grpc/user/user_fiber.pb.go` |
| `protoc-gen-go` (annotations) | `apps/api/handler/grpc/veemon/annotations.pb.go` |
| `protoc-gen-es` | `packages/api-client/src/gen` (TypeScript, shared with `apps/web` and `apps/ai`) |

`user_fiber.pb.go` contains both the route registration and the auth policy map, generated from the same annotation:

```go
// UserApiAuthConfig maps each gRPC full-method name to its auth policy.
var UserApiAuthConfig = map[string]middleware.AuthConfig{
	"/user.UserApi/Register":     middleware.AuthConfig{NeedAuth: false, AllowedRoles: nil},
	"/user.UserApi/Login":        middleware.AuthConfig{NeedAuth: false, AllowedRoles: nil},
	"/user.UserApi/ListUsers":    middleware.AuthConfig{NeedAuth: true, AllowedRoles: []string{"admin", "superadmin"}},
	...
}

func RegisterUserApiRoutes(router v3.Router, srv UserApiServer, validator middleware.TokenValidator) {
	router.Post("/api/v1/auth/login", _UserApi_rateLimit(10, 60*time.Second), _UserApi_Login(srv))
	router.Get("/api/v1/users", middleware.AuthMiddleware(validator, middleware.AuthConfig{NeedAuth: true, AllowedRoles: []string{"admin", "superadmin"}}), _UserApi_ListUsers(srv))
	router.Get("/api/v1/users/:id", middleware.AuthMiddleware(validator, middleware.AuthConfig{NeedAuth: true, AllowedRoles: []string{"admin", "superadmin"}}), _UserApi_GetUser(srv))
}
```

The same map is passed to `middleware.GRPCAuthInterceptor`, so REST and gRPC enforce identical policies.

To add an endpoint: add the RPC plus its `veemon.route` option, run `make -C apps/api proto`, implement the method on `UserApiServer` (`handler/user_handler.go`), then add the usecase and repository methods. No route registration code is written by hand.

## Authentication

Tokens are **PASETO v4 local (symmetric, encrypted)** — not JWT. See `internal/pkg/token/token.go`.

| Function | Purpose |
|----------|---------|
| `token.GenerateSecretKey()` | produce a 32-byte key |
| `token.NewTokenService(secret, expirationHours)` | build the service; rejects weak secrets |
| `TokenService.GenerateToken(userID, email, roles, companyCode)` | issue a token, returning a `jti` claim |
| `TokenService.ValidateToken(tokenString)` | parse, verify, and check `NotExpired` + `ValidAt` |

The key is accepted as a 64-character hex string (32 bytes) or at least 32 raw bytes. Shorter secrets are rejected rather than padded — padding a guessable secret produces a forgeable key. Tokens carry `UserID`, `Email`, `Roles`, `CompanyCode`, and `TokenID` (the `jti` used for revocation).

Send it as a bearer token:

```text
Authorization: Bearer v4.local.xxxxx...
```

### Revocation and lockout

`internal/pkg/authguard/authguard.go` backs two concerns in Redis:

| Function | Redis key |
|----------|-----------|
| `RecordFailure(ctx, email)` / `IsLocked(ctx, email)` | `login:fail:<email>`, `login:lock:<email>` |
| `Reset(ctx, email)` | clears the failure counter |
| `Revoke(ctx, jti, ttl)` / `IsRevoked(ctx, jti)` | `token:revoked:<jti>` |

`provideTokenValidator` in `config/bootstrap.go` composes them: a token that parses but is revoked is rejected with `token.ErrInvalidToken`. Without a Redis client the guard degrades to a no-op rather than failing the request path.

### Fail-closed by design

Auth is never implicit:

- REST routes read their policy from the generated `AuthConfig`. `NeedAuth: false` is an explicit statement that the route is public.
- gRPC uses `middleware.GRPCAuthInterceptor(tokenValidator, pb_user.UserApiAuthConfig)`. A method name absent from the map is **denied** with `no auth policy configured for method` (`internal/pkg/middleware/grpc_auth.go`) rather than served unauthenticated.
- gRPC reflection is registered only when `ENVIRONMENT != "production"`.

Startup also refuses to boot on weak configuration rather than running insecurely — see [Security](#security).

Run `go run ./examples` (from `apps/api`) for an end-to-end issue → call → validate walkthrough.

## Response format

Envelopes live in `internal/pkg/response/response.go`. Protobuf payloads are serialized with `protojson` using `UseProtoNames: false, EmitUnpopulated: true`, so REST output uses the contract's `json_name` (camelCase) fields and unpopulated fields are emitted for a stable shape.

Success (single message, HTTP 200):

```json
{
  "success": true,
  "data": { "id": "...", "name": "...", "email": "..." }
}
```

Created (`RESPONSE_STYLE_CREATED`, HTTP 201) — same body shape.

List (`RESPONSE_STYLE_LIST`, HTTP 200) — the response's single `repeated` field becomes `data`, its pagination message becomes `meta`:

```json
{
  "success": true,
  "data": [{ "id": "...", "name": "..." }],
  "meta": { "page": 1, "size": 20, "total": 137, "totalPages": 7 }
}
```

Error:

```json
{
  "success": false,
  "error": { "code": 400, "message": "invalid request body" }
}
```

Helpers: `SuccessProto`, `CreatedProto`, `SuccessProtoList`, and `BadRequest` / `Unauthorized` / `Forbidden` / `NotFound` / `Conflict` / `InternalError`. The rate limiter's 429 uses the same `success`/`error.code` shape.

## Rate limiting

`internal/pkg/middleware/ratelimit.go` wraps Fiber v3's `middleware/limiter`.

Defaults from `DefaultRateLimitConfig()`:

| Setting | Value |
|---------|-------|
| Max requests | `100` |
| Window | `1 * time.Minute` |
| Key | `c.IP()` |
| Skipped paths | `/health`, `/ready`, `/metrics` |

Exceeding the limit returns HTTP 429 with the standard error envelope.

Per-route limits come from the `veemon.route` `rate_limit` field and are compiled into `_UserApi_rateLimit(max, window)` in `user_fiber.pb.go` — see the login route at `10` requests per `60` seconds above. `RateLimitByEndpointMiddleware` matches limits by `METHOD + " " + path`. `APIKeyRateLimiter` and `UserRateLimiter` key on the API key header or the authenticated user ID, falling back to the IP.

Storage is in-memory unless a `fiber.Storage` is supplied, in which case the limit is shared across replicas.

## Database & migrations

PostgreSQL via GORM. Schema changes go through `golang-migrate`, driven by `cmd/migrate`; it is authoritative — never hand-edit tables to make a migration pass.

Migration files use the golang-migrate convention and live in `apps/api/migrations/`:

```text
migrations/000001_create_users_table.up.sql
migrations/000001_create_users_table.down.sql
```

| Target | Command |
|--------|---------|
| `make migrate` | alias for `migrate-up` |
| `make migrate-up` | run all pending migrations |
| `make migrate-down` | roll back all migrations |
| `make migrate-rollback` | roll back the last migration |
| `make migrate-status` | show the current version |
| `make migrate-create name=<name>` | create a new migration pair |
| `make seed` | run the seeders |
| `make fresh` | drop all, re-migrate |
| `make fresh-seed` | drop all, migrate, seed |
| `make refresh` | roll back all, re-migrate |
| `make refresh-seed` | roll back all, migrate, seed |
| `make reset` | roll back all migrations |

Seeder source lives in `apps/api/database/seeds/` (`database/seeds/seeder.go`); the shared migration driver is `database/migrate`.

## Testing

```bash
make test              # go test -race -count=1 -cover ./...
make test-coverage     # writes coverage.out and coverage.html
```

`make test` runs with the race detector and coverage. Direct invocation:

```bash
go test -race ./...
```

The generated routes have their own tests: `handler/grpc/user/user_fiber_test.go` and `user_grpc_test.go` cover the codegen output as well as your handlers.

## Observability

### Logs

Zap with a JSON encoder (`packages/go-common/logging/logger.go`), keys `timestamp` (ISO 8601), `level`, `message`, plus `caller`. Fields are added as structured key/values — never string-concatenated. Request IDs and trace IDs are attached by the `requestid` and `tracing` middleware. Secrets and PII are never logged.

### Metrics

Prometheus, namespaced by `SERVICE_NAME` (`metrics.Init(cfg.ServiceName)`), with the Go and process collectors registered. Names from `packages/go-common/monitoring/metrics/metrics.go`:

| Metric | Type | Labels |
|--------|------|--------|
| `<ns>_http_requests_total` | counter | `method`, `path`, `status` |
| `<ns>_http_request_duration_seconds` | histogram | `method`, `path`, `status` |
| `<ns>_http_requests_in_flight` | gauge | — |
| `<ns>_http_response_size_bytes` | histogram | `method`, `path` |
| `<ns>_users_registered_total` | counter | — |
| `<ns>_users_logged_in_total` | counter | — |
| `<ns>_active_users` | gauge | — |
| `<ns>_db_queries_total` | counter | — |
| `<ns>_db_query_duration_seconds` | histogram | — |
| `<ns>_db_connections_open` | gauge | — |
| `<ns>_cache_hits_total` | counter | — |
| `<ns>_cache_misses_total` | counter | — |
| `<ns>_messages_published_total` | counter | — |
| `<ns>_messages_consumed_total` | counter | — |
| `<ns>_circuit_breaker_state` | gauge | 0=closed, 1=half-open, 2=open |

`/metrics` is optionally guarded by `metricsAuth(cfg.MetricsAuthToken)`: when `METRICS_AUTH_TOKEN` is set, requests need `Authorization: Bearer <token>`; when unset it is a pass-through and you must restrict it at the network layer.

### Tracing

OpenTelemetry. Set `OTEL_EXPORTER_TYPE` to `otlp`, `stdout`, or `noop` and point `OTEL_ENDPOINT` at a collector. `docker compose up` starts a Jaeger all-in-one on `4317` (OTLP gRPC) and `4318` (OTLP HTTP). The gRPC server attaches `otelgrpc.NewServerHandler()` as its stats handler; the HTTP side is instrumented by the `tracing` middleware.

### Health

| Path | Meaning |
|------|---------|
| `/health` | liveness |
| `/ready` | readiness — checks Postgres, Redis, and RabbitMQ, returning `503` if any is unhealthy and `"disabled"` for an unconfigured dependency |

## Resilience

Outbound work goes through `packages/go-common/infra/resilience`, a failsafe-go executor combining a circuit breaker, a retry policy, and a timeout. Defaults from `DefaultConfig()`:

| Policy | Default |
|--------|---------|
| Circuit breaker — failure threshold | `5` |
| Circuit breaker — success threshold | `3` |
| Circuit breaker — delay | `30s` |
| Circuit breaker — failure rate threshold | `0.5` |
| Retry — initial delay | `100ms` |
| Retry — max delay | `2s` |
| Timeout | `10s` |

Timeout exceeded is logged with the configured duration and surfaces as `resilience.ErrTimeout`. Inbound requests get a separate deadline from the `timeout` middleware.

## Make command reference

Run these from `apps/api`.

### Build and run

| Target | Effect |
|--------|--------|
| `make build` | build `bin/veemon` from `./cmd/server` |
| `make build-worker` | build `bin/veemon-worker` from `./cmd/worker` |
| `make run` | `go run ./cmd/server` |
| `make run-worker` | `go run ./cmd/worker` |
| `make dev` | hot reload via `air` |
| `make infisical-run` | run the server with Infisical-injected secrets |
| `make infisical-run-worker` | run the worker with Infisical-injected secrets |

`INFISICAL_ENV` (default `dev`) and `INFISICAL_PATH` (default `/`) control the `infisical-run*` targets.

### Code

| Target | Effect |
|--------|--------|
| `make proto` | build `protoc-gen-fiber`, run `buf generate` from the repo root |
| `make deps` | `go mod download && go mod tidy` |
| `make fmt` | `go fmt ./...` |
| `make lint` | `golangci-lint run ./...` |
| `make install-tools` | install protoc-gen-go, protoc-gen-go-grpc, buf, air, golangci-lint, gosec |

### Test

| Target | Effect |
|--------|--------|
| `make test` | `go test -race -count=1 -cover ./...` |
| `make test-coverage` | write `coverage.out` and `coverage.html` |
| `make clean` | remove `bin/`, `coverage.out`, `coverage.html` |

### Database

| Target | Effect |
|--------|--------|
| `make migrate` | alias for `migrate-up` |
| `make migrate-up` | run pending migrations |
| `make migrate-down` | roll back all |
| `make migrate-rollback` | roll back the last one |
| `make migrate-status` | current version |
| `make migrate-create name=<name>` | create a migration pair |
| `make seed` | run seeders |
| `make fresh` | drop all, re-migrate |
| `make fresh-seed` | drop all, migrate, seed |
| `make refresh` | roll back all, re-migrate |
| `make refresh-seed` | roll back all, migrate, seed |
| `make reset` | roll back all |

### Docker

| Target | Effect |
|--------|--------|
| `make docker` | build `veemon:latest` |
| `make docker-run` | run the image, publishing `3000` and `50051`, with `--env-file .env` |
| `make compose-up` | `docker-compose up -d` |
| `make compose-down` | `docker-compose down` |

### Release

| Target | Effect |
|--------|--------|
| `make release VERSION=x.y.z` | tag `v<x.y.z>` and push (refuses on a dirty tree) |
| `make release-rc VERSION=x.y.z-rc.1` | tag and push a release candidate |
| `make release-delete VERSION=x.y.z` | delete the tag locally and remotely |
| `make help` | print the target list |

## Security

What the code actually enforces:

- **Fail-closed auth.** No policy means denied, not open — see [Authentication](#authentication).
- **Startup config validation** (`config/config.go`) refuses to start when: `PASETO_SECRET` is unset; `PASETO_SECRET` is still the placeholder `your-secret-key-change-in-production`; `PASETO_SECRET` is not a 64-char hex string or at least 32 bytes; `CORS_ORIGINS` is `*` in production; or `PREFORK` is enabled (unsupported with the embedded gRPC server).
- **Weak PASETO keys rejected.** `NewTokenService` errors on a secret that cannot yield a secure 32-byte key instead of padding it.
- **Password hashing** with `golang.org/x/crypto/bcrypt` at `bcrypt.DefaultCost` (`app/usecase/user/user_usecase.go`). Login compares against a dummy hash for unknown users so response timing does not reveal whether an account exists.
- **Login lockout and token revocation** in Redis — see [Authentication](#authentication).
- **No secrets or PII in logs.** Logging uses structured Zap fields; the repo rules classify data fields and forbid logging secrets.
- **Dependency scanning in CI.** The `vuln` job runs `govulncheck ./...` over `apps/api` and `packages/go-common`.
- **Rate limiting** on credential endpoints — see [Rate limiting](#rate-limiting).

## Project layout

```text
apps/api/
├── cmd/
│   ├── server/          fx.New(ServerModule).Run()
│   ├── worker/          fx.New(WorkerModule).Run()
│   ├── migrate/         migrate CLI: up/down/rollback/status/create/seed/fresh/refresh/reset
│   └── protoc-gen-fiber/  codegen plugin reading veemon.route
├── config/
│   ├── config.go        Viper + Infisical, startup validation
│   ├── fx_*.go          DI modules (CommonModule, ServerModule, WorkerModule)
│   ├── bootstrap.go     constructors + route/health/observability registration
│   └── fiber.go         Fiber app setup
├── handler/
│   ├── user_handler.go  UserApiServer implementation
│   └── grpc/
│       ├── user/        generated .pb.go, _fiber.pb.go, _grpc.pb.go
│       └── veemon/      generated annotations.pb.go
├── app/usecase/user/    business rules
├── repository/user_repository/  GORM queries
├── entity/user.go       plain structs
├── internal/pkg/
│   ├── token/           PASETO v4 issue/verify
│   ├── authguard/       Redis lockout + revocation
│   ├── middleware/      auth, ratelimit, tracing, requestid, timeout, recovery, logger, grpc_*
│   ├── response/        JSON envelopes
│   ├── errors/          typed API errors
│   └── validation/      request validation
├── migrations/          NNNNNN_name.{up,down}.sql
├── database/            migrate driver and seeds/
├── docs/scalar.go       Scalar UI + OpenAPI spec
├── examples/            PASETO authenticated flow example (not part of the build)
├── clients/             generated API clients
├── .env.example
└── Makefile
```

Shared Go infrastructure (logging, telemetry, metrics, Redis, RabbitMQ, resilience, database) lives in `packages/go-common`, a separate module.

## API documentation

`docs/scalar.go` mounts a Scalar API reference backed by an OpenAPI 3.0.3 document, both on the REST port:

| Path | Content |
|------|---------|
| `/docs` | Scalar UI |
| `/docs/*` | Scalar UI |
| `/docs/openapi.json` | the raw OpenAPI document |

The spec is marshalled once at startup and panics there if it is not serializable, so a broken spec fails the boot rather than the first docs request.
