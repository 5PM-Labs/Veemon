# Code Review Checklist

- [ ] All errors are handled properly
- [ ] Context is passed to all I/O operations
- [ ] No goroutine leaks (all goroutines can terminate)
- [ ] Race conditions checked (`go test -race`)
- [ ] Sensitive data is not logged (secrets/PII) — logs go through `logging.*`, not `fmt.Println`
- [ ] New metrics use bounded labels (no free IDs) and live in `internal/infra/monitoring/metrics`
- [ ] Input validation is present
- [ ] Tests are included for new functionality
- [ ] No hardcoded credentials or secrets
- [ ] Database queries use parameterized inputs
- [ ] Resources are properly closed (defer)
- [ ] Every new route/RPC has an explicit auth policy (fail-closed)
- [ ] New env keys are added to `.env.example`
- [ ] REST responses go through `internal/pkg/response` protojson helpers

## Security & compliance (all apps) — SOC 2 / PDPA

- [ ] New data fields are classified ([data-classification](data-classification.md)); PII has a stated purpose + lawful basis
- [ ] No PII/secrets in logs, spans, metric labels, analytics, crash reports, or error messages
- [ ] PII-touching changes are covered by data export + erasure, and retention is declared ([compliance-pdpa](compliance-pdpa.md))
- [ ] Object-level authorization (BOLA) checked in the usecase, not just the route role ([api-security](api-security.md))
- [ ] List endpoints are paginated with a server-side size cap; mass assignment blocked
- [ ] Security-relevant actions write an audit event ([compliance-soc2](compliance-soc2.md) CC7)
- [ ] New third-party service / region → DPA + data inventory updated
- [ ] New dependency justified; lockfile committed; no CI security job skipped ([supply-chain-security](supply-chain-security.md))

## Mobile (`apps/mobile`)

- [ ] Layering holds: Widget → Cubit → UseCase → Repository → DataSource; domain is pure Dart ([flutter-architecture](flutter-architecture.md))
- [ ] Cubits depend on use cases only; states are sealed freezed unions matched with `switch` ([flutter-state-management](flutter-state-management.md))
- [ ] Below-UI APIs return `Either<Failure, T>`; failures shown to the user via `context.l10n` ([flutter-error-handling](flutter-error-handling.md))
- [ ] No string literals in UI (en + id ARB in sync), no hardcoded colors, `context.mounted` after `await`, a11y labels on icon buttons ([flutter-ui](flutter-ui.md))
- [ ] Routes use `AppRoutes`; deep-link params validated ([flutter-navigation](flutter-navigation.md))
- [ ] DTOs match the proto wire shape (camelCase, int64 as string) ([flutter-api-endpoint skill](../skills/flutter-api-endpoint/SKILL.md))
- [ ] DI via `injectable` annotations only; generated files not committed
- [ ] Tokens only in `TokenStorage` (secure storage); nothing sensitive in `--dart-define` ([mobile-security](mobile-security.md))
- [ ] `make -C apps/mobile check` passes (format, analyze = 0 issues, tests)
