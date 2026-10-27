# Data Classification & Handling

Every field this system stores, logs, caches, or sends belongs to exactly one
class. The class — not the developer's judgment in the moment — decides how it
may be handled. [compliance-soc2](compliance-soc2.md) (Confidentiality, Privacy)
and [compliance-pdpa](compliance-pdpa.md) both build on this table. Applies to
**every app** in the monorepo (the Go backend, `apps/web`, `apps/ai`, `apps/mobile`).

## Classes

| Class | Examples in this repo | Who may see it |
|-------|----------------------|----------------|
| **Secret** | `PASETO_SECRET` (PASETO key), DB DSN/password, Redis/RabbitMQ creds, Infisical tokens, model-provider API keys, access tokens, password hashes | Runtime only. No human, no log, no repo. |
| **Restricted PII** (sensitive / "specific" personal data) | Health, biometric, financial/payment, government ID (NIK, NRIC, passport), religion, children's data, precise location | Owning user + explicitly authorized roles, on a need-to-know basis |
| **PII** (personal data) | `email`, `name`, `phone`, IP address, device ID, `user_id` joined to any of the above | Owning user + authorized roles (`admin`, `superadmin`) |
| **Internal** | `user_id` alone, `request_id`, `trace_id`, role names, feature flags, aggregate counts | Engineers and services |
| **Public** | API docs, marketing copy, OpenAPI spec | Anyone |

When unsure, pick the **stricter** class. A new column/field must be classified
in the PR description (see the PR template's *Security & compliance* section).

## Handling matrix

| Action | Secret | Restricted PII | PII | Internal |
|--------|--------|----------------|-----|----------|
| Commit to git / `.env.example` value | ✗ (placeholder only) | ✗ | ✗ (synthetic seed data only) | ✓ |
| Write to logs, spans, metrics labels | ✗ | ✗ | ✗ (use `user_id`) | ✓ (logs/spans; never as a metric label if unbounded) |
| Return in API error message | ✗ | ✗ | ✗ | only if not a 5xx cause |
| Store at rest | KMS / secret manager | DB, encrypted column or app-level encryption | DB (disk encryption) | DB |
| Cache (Redis) | ✗ | ✗ | short TTL, keyed by `user_id` | ✓ |
| Store on a client device | OS keystore only (`flutter_secure_storage`, Tauri secure store) | ✗ unless the feature requires it, then encrypted | minimal, cleared on logout | ✓ |
| Send to a third party (LLM provider, analytics, crash reporter) | ✗ | ✗ without explicit consent + DPA | only with a lawful basis + DPA, minimized | ✓ |
| Use in non-prod (staging/dev) | separate per env | ✗ — synthetic/masked only | ✗ — synthetic/masked only | ✓ |

## Concrete rules

```go
// ✓ Good: correlate by internal ID, never by PII
logging.Warn("failed login attempt",
    zap.String("user_id", userID),          // empty if unknown
    zap.String("email_hash", hashForLog(email)), // keyed HMAC, not plain SHA of the email
    zap.String("ip", c.IP()),
)

// ✗ Bad: PII in logs (also violates observability.md)
logging.Warn("failed login attempt", zap.String("email", email))
```

- **API responses return the minimum.** Never serialize an entity straight to
  the wire — map to the proto message, which lists exactly the allowed fields
  (`UserProfile` has no password hash, no `deleted_at`).
- **Mask on display**, not just on storage: `a***@example.com`, `+62 ***-***-1234`
  in admin UIs unless the operator's task needs the full value.
- **Prompts to `apps/ai` model providers** are a third-party transfer: strip or
  pseudonymize PII before sending, and never send Secret / Restricted PII.
- **Crash reports / analytics** (mobile, web): scrub PII in the SDK's
  `beforeSend` hook; disable automatic screenshot/breadcrumb capture of text fields.

## Retention

Each PII-bearing table/stream declares a retention period in its migration
header comment (see [compliance-pdpa](compliance-pdpa.md) §Retention). Logs that
may contain Internal data: 30–90 days hot; audit logs: at least 1 year
(SOC 2 evidence window), immutable.
