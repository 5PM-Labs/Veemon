# API Security (OWASP API Security Top 10 — 2023)

[security](security.md) covers the general OWASP Top 10. APIs fail in their own
ways; this list is what reviewers check on every route in `contract/*.proto`
and its handler/usecase. Examples are Go (the root module).

## API1 — Broken Object Level Authorization (BOLA)

The #1 API bug: the route is authenticated, but the caller can read/modify
**someone else's object** by changing an ID in the path/body.

```go
// ✓ Good: ownership check in the usecase, for every object fetched by ID
func (uc *UserUseCase) GetUser(ctx context.Context, id string) (*entity.User, error) {
    auth, ok := middleware.AuthFromContext(ctx)
    if !ok {
        return nil, errors.Unauthorized("authentication required")
    }
    isAdmin := slices.Contains(auth.Roles, "admin") || slices.Contains(auth.Roles, "superadmin")
    if auth.UserID != id && !isAdmin {
        return nil, errors.NotFound("user not found") // don't confirm existence
    }
    return uc.repo.FindByID(ctx, id)
}
```

- Role checks in `veemon.route auth { roles }` are **function-level** (API5),
  not object-level — both are needed.
- Use non-guessable IDs (UUIDs) — but never rely on that instead of the check.
- Multi-tenant data (`company_code`) is always filtered by the caller's tenant
  in the repository query, not after loading.

## API2 — Broken Authentication

- PASETO only via `internal/pkg/token`; reject expired/revoked `jti` (`internal/pkg/authguard`).
- Login, register, refresh, and password reset carry a `rate_limit` in the proto
  route and the lockout stays enabled.
- Error messages don't distinguish "unknown email" from "wrong password".
- Tokens travel in `Authorization: Bearer` only — never in query strings
  (they end up in access logs and browser history).

## API3 — Broken Object Property Level Authorization

- **Excessive exposure:** responses are proto messages listing exact fields;
  never marshal a GORM entity to JSON.
- **Mass assignment:** update handlers copy only allowed fields. A normal user
  must not be able to set `status`, `roles`, or `company_code` on themselves
  even if the field exists on `UpdateUserReq` — enforce per role in the usecase.

## API4 — Unrestricted Resource Consumption

- Pagination is mandatory on list RPCs; cap `size` (e.g. ≤ 100) server-side.
- Body size limit on Fiber; request timeout middleware (`internal/pkg/middleware/timeout.go`).
- Bound fan-out (worker pools, see [goroutines](goroutines.md)) and
  outbound-call cost (LLM tokens in `apps/ai`: max tokens + per-user quotas).

## API5 — Broken Function Level Authorization

- Admin RPCs declare `roles: ["admin", "superadmin"]` in the proto — never a
  `if isAdmin` hidden in a handler.
- The generated auth map is fail-closed; keep it that way.

## API6 — Unrestricted Access to Sensitive Business Flows

Flows that are valuable to automate (sign-up, OTP send, checkout, invite)
need abuse controls beyond auth: per-account + per-IP rate limits, CAPTCHA or
device attestation on mobile, and anomaly alerts.

## API7 — Server-Side Request Forgery

Any feature that fetches a user-supplied URL (webhooks, link previews, AI
tools that browse) uses an allowlist + private-IP block (see
[security](security.md) A10), resolves DNS once, and disables redirects to
non-allowed hosts.

## API8 — Security Misconfiguration

- CORS allowlist (no `*` in production — enforced by `Config.Validate()`).
- Security headers on REST responses; no stack traces in error bodies.
- gRPC reflection and Scalar docs disabled or auth-gated in production.

## API9 — Improper Inventory Management

- `contract/` is the inventory: every exposed route is declared there. No
  undocumented debug routes.
- Version breaking changes (`/api/v2`); retire old versions on a schedule.
- Staging/dev APIs never point at production data.

## API10 — Unsafe Consumption of APIs

Treat responses from third-party APIs (payment, LLM, OAuth providers) as
untrusted input: validate schema, enforce timeouts (failsafe-go), cap response
size, and never follow their redirects blindly. LLM output is untrusted —
never execute it or interpolate it into SQL/shell.
