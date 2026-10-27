# Architecture Layers (the Go backend)

The Go module lives at the **repository root** (`go.mod`, module
`github.com/5PM-Labs/veemon`), so one `go build ./...` covers everything.

```
contract/       → Proto contracts (source of truth for gRPC + REST routes)
cmd/server/     → API server entry point (HTTP + gRPC)
cmd/worker/     → RabbitMQ consumer entry point
cmd/migrate/    → migration + seeding CLI
cmd/protoc-gen-fiber/ → Codegen plugin: proto veemon.route → Fiber routes
internal/config/      → Configuration (+ Validate), composition root (fx_*.go), infra init
internal/handler/     → Presentation layer (handlers + generated Fiber routes)
internal/usecase/     → Business logic layer (depends on repository interfaces)
internal/repository/  → Data access layer (pgx, explicit SQL)
internal/domain/      → Domain types + sentinel errors (imported by every layer)
internal/infra/       → Generic infra clients (pgx pool, redis, rabbitmq, logging,
                        telemetry, metrics, resilience, auth0 verification)
internal/pkg/         → App-specific shared code (token, authguard, middleware,
                        response, errors, validation)
internal/docs/        → Scalar API reference + OpenAPI document
migrations/     → golang-migrate SQL files (the schema source of truth)
```

Rules of direction: `handler → usecase → repository → domain`. `internal/infra`
is infrastructure and may be imported by any layer; it never imports
`handler`, `usecase`, or `repository`. `internal/pkg` is presentation/auth
coupled and stays app-only.

**One Go module, no `replace`.** `internal/infra` replaced the old
`packages/go-common` module, so Docker, goreleaser, CI, and `go build` all run
from the repo root with no cross-module path juggling. `apps/` and `packages/`
now hold only the TS/Flutter workspaces.

**Postgres access is pgx, not an ORM.** Repositories write explicit SQL against
`*pgxpool.Pool`; row scanning goes through one shared `scanUser` helper because
pgx refuses NULL into a plain string. Failed writes are translated to
`domain.ErrDuplicateEntry` / `domain.ErrNotFound` at the repository boundary so
no caller imports the driver. Multi-statement writes use `database.InTx`.

**Composition root:** `cmd/server/main.go` and `cmd/worker/main.go` are thin
`fx.New(...).Run()` calls — all wiring (repository → usecase → handler →
routes, infra init, and lifecycle-managed start/stop) lives in
`internal/config/fx_*.go` (`CommonModule`, `ServerModule`, `WorkerModule`).

**REST routes are generated, not hand-written.** Declare a route with a
`veemon.route` option on the RPC in `contract/<svc>/<svc>.proto` (method, path,
`auth { required, roles, permissions }`, `body`, `response`, `rate_limit`), then
run `make proto`. `protoc-gen-fiber` emits
`internal/handler/grpc/<svc>/<svc>_fiber.pb.go` (`Register<Svc>Routes` + the
`<Svc>AuthConfig` gRPC auth map). Never add a Fiber route or auth map by hand.
See the [proto-routes skill](../skills/proto-routes/SKILL.md).

## Authorization (RBAC)

- `roles`, `permissions`, `role_permissions`, `user_roles` tables are the
  authorization source of truth. **`user_roles` — not a column on `users` — is
  authoritative**, so a role change takes effect on the caller's next request
  with no re-login.
- Permission codes are `<resource>:<action>` (`users:read`, `roles:update`, …);
  `*` grants everything and is held by `superadmin` only.
- The validator resolves roles + permissions per request and puts them on
  `middleware.AuthContext`. `AuthConfig.AllowedRoles` and `AllowedPerms` are
  OR-ed; an empty policy after `NeedAuth` means "any authenticated caller".
- Deny by default: a route with no policy fails closed in the gRPC interceptor.

## Data Flow (both protocols share the same handler + usecase)

```
HTTP  → generated Fiber routes (handler/grpc/<svc>) ─┐
                                                      ├→ handler → UseCase → Repository → Postgres
gRPC  → <Svc>Api server ─────────────────────────────┘
```

REST and gRPC share one handler implementation and one auth policy map, so the
two protocols can never drift.
