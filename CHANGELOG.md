# Changelog

All notable changes to this boilerplate are documented here.

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

## [Unreleased] — Reliability & Security Hardening

A comprehensive hardening pass across correctness, security, reliability,
concurrency, observability, packaging, and docs.

### Critical fixes
- **Roles column now scans correctly.** `entity.User.Roles` is `pq.StringArray`
  (was a plain `[]string` mapped to `text[]`, which GORM/pgx could not scan —
  every authenticated read failed against real Postgres).
- **Forgeable default token secret removed.** `JWT_SECRET` has no default; the
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
