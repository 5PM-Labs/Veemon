# Claude Code Guidelines for Veemon

This file is the **index** for AI-assistant guidance. The detailed standards,
rules, agents, and skills live under [`.claude/`](.claude/) so each concern is a
small, focused file. Read the rule that matches the task; treat these as
authoritative for `apps/api` (the Go backend), the Flutter rules for
`apps/mobile`, and the security & compliance rules for every app.

> This repo is a **polyglot Bun-workspace monorepo** — a Go backend (`apps/api`),
> a React + Tauri client (`apps/web`), a Flutter mobile client (`apps/mobile`), and a
> LangGraph.js service (`apps/ai`), sharing one proto contract (`contract/`). Start with
> [rules/project-overview.md](.claude/rules/project-overview.md).

## Rules (`.claude/rules/`)

| Rule | Covers |
|------|--------|
| [project-overview](.claude/rules/project-overview.md) | Stack, monorepo layout, root Bun scripts |
| [architecture](.claude/rules/architecture.md) | Go backend layers, generated routes, data flow |
| [codebase-conventions](.claude/rules/codebase-conventions.md) | Fail-fast config, fail-closed auth, protojson responses |
| [coding-standards](.claude/rules/coding-standards.md) | File org, naming, errors, context, interfaces |
| [goroutines](.claude/rules/goroutines.md) | Leaks, races, deadlocks, worker pools, panics |
| [database](.claude/rules/database.md) | GORM usage, transactions, connection pool |
| [redis](.claude/rules/redis.md) | Redigo connection pooling |
| [testing](.claude/rules/testing.md) | Table-driven tests, mocks, race detection |
| [observability](.claude/rules/observability.md) | Logging shape, metric/span conventions, correlation, no secrets/PII |
| [security](.claude/rules/security.md) | OWASP Top 10 with Go examples |
| [api-responses](.claude/rules/api-responses.md) | Success / error / pagination envelopes |
| [commands](.claude/rules/commands.md) | Common `make` / `bun` commands |
| [code-review-checklist](.claude/rules/code-review-checklist.md) | Pre-PR checklist |

### Security & compliance — mandatory for every app

| Rule | Covers |
|------|--------|
| [compliance-soc2](.claude/rules/compliance-soc2.md) | SOC 2 Trust Services Criteria → access control, audit log, change mgmt, availability, encryption (+ ISO 27001 crosswalk) |
| [compliance-pdpa](.claude/rules/compliance-pdpa.md) | PDPA (SG/MY/TH) + UU PDP (ID): consent, data-subject rights, retention, cross-border, breach notification |
| [data-classification](.claude/rules/data-classification.md) | Secret / Restricted PII / PII / Internal / Public and how each may be logged, stored, shared |
| [api-security](.claude/rules/api-security.md) | OWASP API Security Top 10 (2023) — BOLA, mass assignment, resource limits |
| [supply-chain-security](.claude/rules/supply-chain-security.md) | Secrets, dependency/vuln scanning, CI hardening, SBOM, release signing |
| [mobile-security](.claude/rules/mobile-security.md) | OWASP MASVS v2 for `apps/mobile` |

### Mobile — `apps/mobile` (Flutter)

Scoped with `paths: apps/mobile/**` frontmatter — they load only when working
on mobile files.

| Rule | Covers |
|------|--------|
| [flutter-architecture](.claude/rules/flutter-architecture.md) | Feature-first Clean Architecture, layer direction, injectable DI, data layer, file structure |
| [flutter-state-management](.claude/rules/flutter-state-management.md) | Cubits call use cases only, read vs write cubits, freezed sealed states, `switch` patterns, async transitions, narrow rebuilds |
| [flutter-ui](.claude/rules/flutter-ui.md) | Private widget classes, `const`, theme-only colors, `context.l10n`, `context.mounted`, responsive, accessibility, forms, performance |
| [flutter-error-handling](.claude/rules/flutter-error-handling.md) | `Either<Failure, T>`, `guard()`, specific `Failure` types, localized messages, never hide errors, logging |
| [flutter-navigation](.claude/rules/flutter-navigation.md) | go_router constants, auth via `redirect`, `go` vs `push`, deep-link param validation |
| [flutter-release](.claude/rules/flutter-release.md) | `--dart-define` config, versioning, release builds, signing, native config, store compliance |
| [flutter-coding-standards](.claude/rules/flutter-coding-standards.md) | Analyzer (0 issues), naming, imports, Dart idioms, generated code, dependencies, workflow |
| [flutter-testing](.claude/rules/flutter-testing.md) | Use case / cubit / repository / widget tests, mockito + bloc_test, must-have tests per change |

## Agents (`.claude/agents/`)

- [code-reviewer](.claude/agents/code-reviewer.md) — reviews Go changes against these rules before a PR.
- [mobile-reviewer](.claude/agents/mobile-reviewer.md) — reviews `apps/mobile` (Flutter) changes against the Flutter + MASVS rules.

## Skills (`.claude/skills/`)

- [proto-routes](.claude/skills/proto-routes/SKILL.md) — add/change a REST+gRPC endpoint via the proto contract, then `make proto`.
- [flutter-feature](.claude/skills/flutter-feature/SKILL.md) — scaffold a feature/screen/use case in `apps/mobile` (domain → data → presentation → DI → route → tests).
- [flutter-api-endpoint](.claude/skills/flutter-api-endpoint/SKILL.md) — call a `contract/*.proto` route from `apps/mobile` (DTO wire shape, endpoint, data source, `guard()` mapping, tests).

## Quick rules of thumb

- **REST routes are generated** — change `contract/<svc>/<svc>.proto` and run
  `make proto`; never hand-edit Fiber routes or the auth map.
- **Auth is fail-closed** — every route/RPC needs an explicit policy or the server
  panics at startup.
- **golang-migrate is authoritative** — schema changes go through SQL migrations.
- **Always** `go test -race ./...` for new behavior; never log secrets.
- **SOC 2 + PDPA are mandatory** — classify new data fields, keep PII out of
  logs/analytics, audit-log security events, never bypass CI/review.
- **Mobile is Clean Architecture** — Widget → Cubit → UseCase → Repository →
  DataSource; `Either<Failure, T>` below the UI; `make -C apps/mobile check`.
- **Composition root, not `main()`** — `cmd/server`/`cmd/worker` are thin
  `fx.New(...).Run()` entry points; wiring lives in `config/fx_*.go`
  (`CommonModule`/`ServerModule`/`WorkerModule`). `apps/ai`'s `composition.ts`
  uses Awilix the same way. Shared Go infra (logging, telemetry/metrics,
  redis/rabbitmq/resilience/database) lives in `packages/go-common` — a
  separate module, not `apps/api/pkg`.

## Related docs

- [README.md](README.md) — full project documentation.
- [AGENTS.md](AGENTS.md) — same guidance, for agent tools that read `AGENTS.md`.
