# SOC 2 — Engineering Controls (mandatory)

SOC 2 is an attestation against the AICPA **Trust Services Criteria (TSC)**. The
auditor does not read our code; they test whether **controls exist and operate
continuously** and ask for **evidence**. This rule lists the controls that live
in the codebase and the delivery pipeline, so every change keeps them working.
Applies to all apps; Go examples target `apps/api`.

> Type I = controls designed correctly at a point in time. Type II = controls
> *operated* over a period (usually 6–12 months). Assume **Type II**: a control
> that is skipped "just this once" is an audit exception.

## Criteria in scope

| TSC | Meaning | Primary rules |
|-----|---------|---------------|
| **Security (CC1–CC9)** — always in scope | Protection against unauthorized access | this file, [security](security.md), [api-security](api-security.md), [supply-chain-security](supply-chain-security.md) |
| **Availability (A1)** | System is up per commitments | Backups, health checks, capacity, DR (below) |
| **Confidentiality (C1)** | Confidential data protected through its lifecycle | [data-classification](data-classification.md) |
| **Processing Integrity (PI1)** | Processing is complete, valid, accurate | Validation, idempotency, transactions ([database](database.md)) |
| **Privacy (P1–P8)** | Personal info handled per notice | [compliance-pdpa](compliance-pdpa.md) |

## CC6 — Logical access

- **Fail-closed authorization** on every route/RPC (already enforced:
  `mustAuthConfig` panics on a missing policy; gRPC interceptor denies unknown
  methods). Never add an "allow all" escape hatch.
- **Least privilege:** new roles/permissions are added to the proto
  `auth { roles }` — not checked ad hoc in handlers. Object-level ownership
  checks live in the usecase (see [api-security](api-security.md) API1).
- **Session control:** tokens are short-lived PASETO with a revocable `jti`
  (`pkg/authguard`). Logout and password change **must** revoke. Login lockout
  stays on.
- **Production access:** no shared accounts; DB/infra access via SSO + MFA,
  time-bound, and logged. No production credentials on developer laptops or in
  CI logs.
- **Access reviews:** roles granting `admin`/`superadmin` are reviewed
  quarterly — keep role assignment queryable (no roles hidden in config files).

## CC7 — System operations: logging, monitoring, incident response

**Audit log (required, currently a gap).** Security-relevant events must be
written to an append-only audit trail, separate from debug/app logs:

| Event | Fields |
|-------|--------|
| login success / failure / lockout | `actor_id` (or `email_hash`), `ip`, `user_agent`, `result` |
| token refresh / logout / revocation | `actor_id`, `jti` (not the token) |
| role or permission change | `actor_id`, `target_id`, `before`, `after` |
| admin read/update/delete of another user's data | `actor_id`, `target_id`, `action` |
| data export / erasure request (PDPA) | `actor_id`, `subject_id`, `action`, `ticket` |
| config/secret rotation, migration run | `actor`, `version` |

```go
// Shape: structured, no PII beyond IDs, correlated by request_id.
auditLog.Record(ctx, audit.Event{
    Action:   "user.update",
    ActorID:  authCtx.UserID,
    TargetID: targetID,
    Result:   audit.ResultSuccess,
    Metadata: map[string]string{"fields": "name,status"}, // field names, never values
})
```

Audit records are immutable (no UPDATE/DELETE grants for the app role; ship to
write-once storage), retained **≥ 1 year**, and their write failure is an
ERROR + metric, never silently dropped.

- **Monitoring/alerting:** 5xx rate, auth-failure spikes, lockout spikes,
  circuit-breaker opens, and queue backlog have alerts (metrics in
  `packages/go-common/monitoring/metrics`).
- **Incident response:** a security incident follows the runbook (triage →
  contain → eradicate → recover → post-mortem). If personal data is involved,
  the PDPA breach clock starts at assessment — see
  [compliance-pdpa](compliance-pdpa.md) §Breach notification.

## CC8 — Change management

Every production change is traceable: **ticket → PR → review → CI → deploy**.

- No direct pushes to `main`; branch protection requires ≥ 1 approving review
  that is **not the author** and green CI (lint, test, vuln scan, build).
- PR uses the template; the *Security & compliance* section is filled in, not
  deleted.
- Schema changes only through golang-migrate files (reviewed like code).
- Emergency fixes still go through a PR; review may happen post-merge within
  1 business day and is recorded.
- Releases are tagged and reproducible (see `.github/workflows/release.yml`).

## CC9 / Vendor risk

A new third-party service that receives our data (LLM provider, crash reporter,
email/SMS gateway, analytics) needs: a security review, a DPA, the data classes
it receives recorded in the PR, and an entry in the subprocessor list. See
[supply-chain-security](supply-chain-security.md) for code dependencies.

## A1 — Availability

- `/health` (liveness) and `/ready` (readiness: DB, Redis, RabbitMQ) endpoints
  stay accurate — readiness fails when a hard dependency is down.
- Graceful shutdown via `fx.Lifecycle` `OnStop` — drain in-flight requests.
- Postgres: automated backups + point-in-time recovery, **restore tested** at
  least quarterly (an untested backup is not a control). Define RPO/RTO.
- Resilience on outbound calls (failsafe-go timeout/retry/circuit breaker);
  every outbound call has a timeout.

## PI1 — Processing integrity

- Validate all input at the boundary (`pkg/validation`) — reject, don't coerce.
- Multi-step writes in a transaction; message consumers are **idempotent**
  (RabbitMQ redelivers).
- Column-scoped updates (no lost updates) — see [codebase-conventions](codebase-conventions.md).

## Encryption (CC6.1 / CC6.7)

- In transit: TLS 1.2+ everywhere external; internal service hops TLS where
  the network is not fully trusted. No `InsecureSkipVerify` outside tests.
- At rest: managed-disk encryption for DB/Redis/backups; Restricted PII
  additionally app-level encrypted with a KMS-managed key.
- Keys/secrets: from the secret manager (Infisical/env), rotated on a
  schedule and immediately on suspected exposure. `Config.Validate()` rejects
  weak/placeholder values.

## Evidence the pipeline already produces — keep it

PR history + reviews, CI runs (lint/test/govulncheck), release tags, migration
history. Do not disable, bypass, or `continue-on-error` a CI security job to
merge faster — that destroys evidence for the whole audit period.

## ISO 27001 crosswalk

The same controls satisfy ISO/IEC 27001:2022 Annex A — mainly A.5.15–5.18
(access), A.8.15–8.16 (logging/monitoring), A.8.24 (cryptography), A.8.25–8.32
(secure development, change management), A.5.24–5.28 (incident management).
