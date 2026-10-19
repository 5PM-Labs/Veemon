# Codebase Conventions (established — follow these)

- **Config validates and fails fast.** `Config.Validate()` (called in
  `cmd/server`) rejects a weak/placeholder `PASETO_SECRET`, `PREFORK=true` (the
  embedded gRPC server is incompatible with Fiber prefork), and `CORS_ORIGINS=*`
  in production. Never ship a usable default secret; add new env keys to
  `.env.example` and bind them (all keys are bound via reflection so `Unmarshal`
  reads env-only overrides).
- **Auth is fail-closed.** Every route/RPC must have an explicit policy in
  `handler/grpc/user` (`RouteAuthConfig` / `AuthConfigMethods`). REST uses
  `mustAuthConfig(...)`, which panics at startup if a route has no policy; the
  gRPC interceptor denies unknown methods. Adding an endpoint without a policy is
  a startup crash, not a silent exposure.
- **Auth context uses typed keys.** Use `middleware.WithAuthContext` /
  `middleware.AuthFromContext` — never `ctx.Value("auth")`.
- **golang-migrate is authoritative.** Change schema via SQL migrations;
  `AutoMigrate` is opt-in (`DB_AUTO_MIGRATE`, default off) for local dev only.
- **Updates are column-scoped.** Use `Updates(map/struct)` on the changed columns
  (see `repository.UpdateFields`), not read-modify-write with `Save`, to avoid
  lost updates.
- **REST responses go through `internal/pkg/response` protojson helpers** (`SuccessProto`,
  `CreatedProto`, `SuccessProtoList`) so field names are camelCase per the proto
  contract.
- **5xx causes are logged, never leaked.** Map domain errors to `internal/pkg/errors`
  helpers; log the underlying cause server-side and return a sanitized message.
