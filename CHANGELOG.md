# Changelog

All notable changes to this boilerplate are documented here.

## [Unreleased] — Repo restructure, pgx, RBAC, Auth0

### Changed
- **Single Go module at the repo root.** `apps/api/go.mod` and the separate
  `packages/go-common` module are gone; `go.mod` sits at the root (module
  `github.com/5PM-Labs/veemon`) and `internal/infra` replaces `veemon-common`.
  Docker, goreleaser, CI, and `make` all run from the root — no cross-module
  `replace`, no per-app Go build.
- **Layout.** `apps/api/{cmd,config,handler,entity,repository,app,docs,migrations}`
  → `cmd/`, `internal/{config,handler,domain,repository,usecase,docs}`,
  `migrations/`. `internal/` keeps app code unimportable from outside.
- **GORM → pgx v5.** Repositories are explicit SQL over `*pgxpool.Pool`;
  `gorm`/`gorm.io/*` are gone. Driver errors are translated at the repository
  boundary (`domain.ErrDuplicateEntry`, `domain.ErrNotFound`); multi-statement
  writes use `database.InTx`.
- **Root `package.json`** points its Go scripts at root `make` targets.

### Added
- **RBAC.** `roles`, `permissions`, `role_permissions`, `user_roles`, and
  `audit_logs` (migration `000002`), seeded with `superadmin`/`admin`/`user`.
  Routes declare `auth { permissions: [...] }` in the proto; the validator
  resolves roles + permissions per request, so a role change applies on the
  next request without re-login. `*` is the superadmin wildcard.
- **Auth0 login.** `POST /api/v1/auth/auth0/login` verifies an Auth0 RS256
  token against the tenant JWKS and issues a first-party PASETO token,
  provisioning the account (`authProvider=auth0`) on first login. Off unless
  `AUTH0_DOMAIN` + `AUTH0_AUDIENCE` are set.
- **Detailed auth columns** on `users`: `avatar_url`, `locale`, `timezone`,
  `email_verified`, `mfa_enabled`, `auth_provider`, `auth0_subject`, `metadata`
  (JSONB), `last_login_at`, `failed_login_count`, `locked_until`.
- **CI**: Postgres service job that applies the real migrations and runs the
  `-tags integration` repository tests; `gofmt` gate; dependency-review via
  Dependabot (Go, npm, Actions, Docker). Docker images build on demand only
  (`.github/workflows/docker.yml`), not on every push.

### Removed
- `users.roles` text[] — `user_roles` is the single source of truth, so a role
  list can never disagree with the RBAC tables.
- `apps/api/` and `packages/go-common/` directories.

## [Unreleased] — Hardening pass

### Security
- **Pinned JS runtime** — CI no longer installs `bun-version: latest`; the Bun
  version is `BUN_VERSION` (1.4.3) in `.github/workflows/ci.yml`, mirrored by
  `.tool-versions`. Bun vendors Node code (undici, c-ares, TLS, brotli), so the
  Node 2026-10-21 security release (CVE-2026-105481…111497) is addressed by
  tracking a patched Bun build rather than adding a Node install step. The repo
  runs no Node process of its own.

### Fixed
- **Pagination** — omitting `page`/`size` on `GET /api/v1/users` produced
  `LIMIT 0` and a division by zero; values are now clamped at the usecase
  (default `size=10`, max `100`) and echoed in the response.
- **Rate limits** — per-route and global limits use Redis storage when
  available, so the 10/min login limit holds across replicas instead of per
  process.
- **Auth fail-closed** — `GetMe`/`RefreshToken`/`Logout` share one guard that
  returns 401 when no auth context is present. Removed the panic-based
  `MustGetAuthContext`.
- **API docs** — the Scalar UI is no longer served when `ENVIRONMENT=production`
  (same rule as gRPC reflection).
- **`apps/ai`** — startup fails in production without `API_SERVICE_TOKEN`, and
  request errors are no longer echoed to clients (validation messages only).
- **Schema drift** — removed GORM `uniqueIndex`/`default` tags that conflicted
  with the partial unique index in `000001`; fixed the down migration.

### Changed
- Environment variables `JWT_SECRET` → `PASETO_SECRET`,
  `JWT_EXPIRATION` → `PASETO_EXPIRATION_HOURS`.
- `DB_AUTO_MIGRATE` is rejected in production.
- `protoc-gen-fiber` emits `RegisterXxxRoutes(router, srv, validator, redisClient)`.

### Added
- `lefthook.yml` — Go format/lint and Dart format on `pre-commit`, tests on
  `pre-push`.

## [Unreleased] — Compliance rules & Flutter mobile app

### Added
- **Security & compliance rules** (`.claude/rules/`): `compliance-soc2`
  (Trust Services Criteria → engineering controls, ISO 27001 crosswalk),
  `compliance-pdpa` (SG/MY/TH PDPA + Indonesia UU PDP), `data-classification`,
  `api-security` (OWASP API Top 10 2023), `supply-chain-security`,
  `mobile-security` (OWASP MASVS v2). PR template gains a *Security &
  compliance* section; review checklist and `code-reviewer` agent extended.
- **`apps/mobile`** — Flutter iOS + Android client, feature-first Clean
  Architecture (Cubit, freezed 3, injectable/get_it, fpdart `Either`, go_router,
  l10n en/id). Auth feature end-to-end against the Go API (login, session
  restore via `/auth/me`, logout), token in secure storage, Android/iOS
  network + backup hardening. Rules `flutter-architecture`,
  `flutter-coding-standards`, `flutter-testing`; `mobile-reviewer` agent; CI
  `mobile` job (gen, format, analyze, test).

### Changed
- `security.md` A09 example no longer logs a raw email (PII).
- **Dependencies updated across all apps** (minor/patch only, no majors):
  - Go toolchain **1.25.12 → 1.26.9** (CI, release, Dockerfile, `go.mod`).
    Go 1.25 is out of support since Go 1.27, `validator` v10.30.5 requires
    Go ≥ 1.26, and go1.26.9 fixes a `net/http` vuln that's unfixed on 1.25.
  - `apps/api` + `packages/go-common`: fiber v3.6.0, grpc v1.84.0, otel
    v1.47.0 (incl. OTLP exporters), golang-migrate v4.20.1, x/crypto v0.58.0,
    amqp091-go v1.15.0, gorm postgres v1.6.3, testify v1.12.1, and more —
    clears the govulncheck findings (otel SDK/exporters < 1.45, grpc < 1.83.1,
    stdlib < 1.25.13).
  - `apps/web`: react 19.3, vite 8.3, TanStack Router/Query, Tauri CLI 2.12.
  - `apps/ai`: LangChain/LangGraph 1.x patches, hono 4.13, zod 4.6.
  - `apps/mobile`: transitive upgrades; `freezed` 4.0.2 / `build_runner`
    2.16.2 held back until `injectable_generator` supports the newer analyzer.

## [Unreleased] — Reliability & Security Hardening

A comprehensive hardening pass across correctness, security, reliability,
concurrency, observability, packaging, and docs.

### Critical fixes
- **Roles column now scans correctly.** `entity.User.Roles` is `pq.StringArray`
  (was a plain `[]string` mapped to `text[]`, which GORM/pgx could not scan —
  every authenticated read failed against real Postgres).
- **Forgeable default token secret removed.** `PASETO_SECRET` has no default; the
  server refuses to start if it is empty, the old placeholder, or weaker than a
  32-byte / 64-hex key. Short secrets are rejected, not silently padded.

### Security
- Fail-closed auth wiring: a route/RPC with no explicit policy is denied (REST
  panics at startup on a missing policy; gRPC rejects unknown methods).
- Global + stricter per-auth-route rate limiting; Redis-backed account lockout.
- Token revocation via a `jti` denylist — logout revokes, refresh rotates and
  reloads the user; `authguard` degrades to a no-op without Redis.
- Login rejects non-active accounts; user-enumeration timing mitigated.
- Password complexity enforced (upper+lower+digit, ≤72 bytes for bcrypt).
- Helmet security headers; CORS wildcard blocked in production; gRPC reflection
  disabled in production.

### Reliability
- Auto-reconnecting RabbitMQ client with per-consumer channels, poison-message
  handling, and panic recovery; worker drains in-flight messages on shutdown.
- HTTP server read/write/idle timeouts; per-request context deadline; Redis
  dial/read/write timeouts; configurable DB connection pool.
- gRPC recovery/logging/tracing interceptors; ordered graceful shutdown
  (HTTP drain → gRPC GracefulStop with timeout → DB close).
- `/ready` actually pings Redis and RabbitMQ. `PREFORK` is rejected (it is
  incompatible with the embedded gRPC server).

### Correctness
- Duplicate-email race maps to 409 (via `gorm:TranslateError`); UUID path params
  validated; ORDER BY column whitelist; column-scoped updates (no lost updates).
- REST responses use `protojson` (camelCase, matching the proto contract).
- golang-migrate is the single source of truth; AutoMigrate is opt-in
  (`DB_AUTO_MIGRATE`, default off). Users email is uniquely indexed only among
  non-deleted rows so deleted emails can be reused.

### Concurrency
- Rate-limiter map pre-built (fixes a concurrent-map-write crash); Fiber
  zero-copy strings copied before use in spans; Infisical refresh goroutine no
  longer leaks; resilient HTTP client buffers the body for retries; global
  logger init is concurrency-safe.

### Observability
- Non-blocking OTLP exporter; real `noop` exporter; configurable sampler.
- Access logs and metrics record the true status (including errors and panics);
  route-pattern span names / metric labels; request-id ↔ trace correlation.
- `/metrics` supports optional bearer-token protection; GORM logger no longer
  logs interpolated SQL by default.

### Build / CI / packaging
- CI runs `golangci-lint` (v2 config) + tests with `-race` + `govulncheck`, and
  validates the Docker build on PRs.
- Multi-stage Dockerfile ships server + migrate + worker, runs as non-root, adds
  a healthcheck and `.dockerignore`; compose runs migrations before the app.
- GoReleaser builds all three binaries; Go version aligned to 1.25.11 everywhere.

### Removed
- Dead `pkg/jwt` (superseded by PASETO `pkg/token`), dropping the `golang-jwt`
  dependency.
- Orphaned `payslips` and `audit_logs` migrations and stale docs.
