# Project Overview & Monorepo Layout

This repo is a **polyglot Bun-workspace monorepo**. The Go backend lives in
the repo-root Go module (all Go paths in these rules are relative to it); `apps/web` is a
React + TanStack Router + Tauri client and `apps/ai` is a LangGraph.js service, both
consuming the Go API through the generated `@veemon/api-client`
(`packages/api-client`). `apps/mobile` is a Flutter (iOS + Android) client that
calls the same REST routes through its own Dart DTOs (see
[flutter-architecture](flutter-architecture.md)). The proto contract in `contract/` (repo root) is the
single source of truth generating Go **and** TypeScript. See the layout below and
`docs/superpowers/specs/` for the design.

The Go backend (root module: `cmd/`, `internal/`) is a Go monolithic application using:

- **Go Fiber** for REST API
- **gRPC** for service-to-service communication
- **Domain-Driven Design (DDD)** with Clean Architecture
- **GORM** for database ORM (PostgreSQL); **golang-migrate** for schema migrations (source of truth)
- **PASETO v4** (`internal/pkg/token`) for authentication tokens; **Redis-backed** login lockout + token revocation (`internal/pkg/authguard`)
- **Redigo** for Redis caching
- **RabbitMQ** for message queuing (auto-reconnecting client)
- **Zap** for structured logging
- **OpenTelemetry** for distributed tracing
- **Viper** for configuration management
- **go-playground/validator** for validation
- **failsafe-go** for resilience (circuit breaker, retry, timeout)
- **Prometheus** for metrics and monitoring
- **pgx/v5** for Postgres (explicit SQL — no ORM)
- **Scalar** for API documentation (OpenAPI 3)
- **uber-go/fx** for dependency injection — `cmd/server`/`cmd/worker` are thin
  `fx.New(...).Run()` composition roots (see `internal/config/fx_*.go`)
- **RBAC** — `roles`/`permissions`/`user_roles`, permission-based route policies

## Monorepo Layout

```
cmd/            → server · worker · migrate · protoc-gen-fiber entry points
internal/       → Go backend: config, handler, usecase, repository, domain,
                  infra, pkg, docs — the module the rules describe
migrations/     → golang-migrate SQL (schema source of truth)
apps/web/       → React + TanStack Router + Vite + Tauri (@veemon/web)
apps/ai/        → LangGraph.js agent + workflow service (@veemon/ai, Hono HTTP)
apps/mobile/    → Flutter iOS + Android client — Clean Architecture + Cubit
                  (make-driven, not a Bun workspace member)
packages/api-client/ → generated protobuf-es types + typed REST client (@veemon/api-client)
packages/tsconfig/   → shared base tsconfig (@veemon/tsconfig)
go.mod          → single Go module at the root (no per-app module, no `replace`)
contract/       → proto source of truth (repo root) — generates Go + TS
buf.gen.yaml    → Go → internal/handler/grpc; TS → packages/api-client/src/gen
package.json    → Bun workspaces + root scripts (proto, dev, build, test)
```

Root scripts (Bun): `bun run proto` (regenerate Go + TS), `bun run dev`
(api + web + ai), `bun run build`, `bun run test`. The Go backend and `apps/mobile` are
orchestrated via `make` and are not Bun workspace members. The TS apps call the existing Fiber
REST routes through `@veemon/api-client` — no changes to the Go server (Approach A).
