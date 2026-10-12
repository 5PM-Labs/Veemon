---
paths:
  - "apps/mobile/**"
---

# Flutter Error Handling (`apps/mobile`)

Two error worlds, one boundary:

```
data layer:     throws AppException  (NetworkException, RequestTimeoutException,
                                      ApiException, MalformedResponseException)
                        │
            RepositoryImpl + guard()  ← the ONLY place exceptions become failures
                        ▼
domain + UI:    Either<Failure, T>   (never throws)
```

Code: `lib/core/error/{exceptions,failures,guard,failure_message}.dart`.

## 1. Everything below the UI returns `Either<Failure, T>`

```dart
// ✓ Good: domain/repositories/auth_repository.dart
Future<Either<Failure, UserEntity>> getCurrentUser();
Future<Either<Failure, Unit>> logout();        // Unit, not void

// ✗ Bad: callers can't see it fails, and must remember to try/catch
Future<UserEntity> getCurrentUser();
```

- Use cases and repository interfaces never throw and never return nullable
  "error = null" values.
- Use `fpdart`: `Right(value)`, `Left(failure)`, `unit`, `result.map(...)`,
  `result.getLeft()`. Match with `switch` (see
  [flutter-state-management](flutter-state-management.md) §5).

## 2. Repositories convert with `guard()`

```dart
// ✓ Good: data/repositories/auth_repository_impl.dart
@override
Future<Either<Failure, UserEntity>> login({required String email, required String password}) => guard(
  () async {
    final response = await _remote.login(email: email, password: password);
    await _tokenStorage.write(response.token);
    return response.user.toEntity();
  },
  // Endpoint-specific meaning: on /login, a 401 is bad credentials, not an expired session.
  onApiException: (e) => e.statusCode == 401 ? const Failure.invalidCredentials() : null,
);
```

- `guard()` maps the defaults: 401 → `unauthorized`, 429 → `rateLimited`,
  other HTTP → `server(statusCode, code)`, offline → `network`,
  timeout → `timeout`, contract mismatch → `unexpected`.
- Use `onApiException` for endpoint-specific meanings. Return `null` to fall
  back to the default mapping.
- Data sources throw `AppException` subtypes only. Wrap parser errors as
  `MalformedResponseException` (see `_parse` in `auth_remote_data_source.dart`).
- Never `catch (e) {}` and never `catch (e) { return Left(Failure.unexpected()); }`
  around everything. `guard` only catches `AppException`, so real bugs still
  crash loudly in debug.

## 3. Specific `Failure` types

`core/error/failures.dart` is a freezed sealed union:

| Failure | When | What the UI does |
|---------|------|------------------|
| `network()` | offline / unreachable / TLS | "You appear to be offline", allow retry |
| `timeout()` | connect/send/receive timeout | "Took too long", allow retry |
| `unauthorized()` | missing/expired/revoked session | sign out → login |
| `invalidCredentials()` | login 401 | inline error on the form |
| `rateLimited()` | 429 / lockout | "Too many attempts, wait" |
| `validation(ValidationIssue)` | use-case input rule | inline field error |
| `server(statusCode, code)` | other non-2xx | generic "our side" message |
| `unexpected()` | contract mismatch / bug | generic message, log it |

Add a new subtype **only** when the UI must react differently, and then
update `failure_message.dart` and both ARB files in the same PR. The
exhaustive `switch` there makes a missing message a compile error. Don't add
a `message: String` field "for flexibility".

## 4. Users see localized, actionable messages

```dart
// ✓ Good
Text(failure.message(context))   // FailureMessageX → context.l10n.errorX

// ✗ Bad: leaks technical detail, and isn't translated
Text(e.toString())
Text('Error ${failure.statusCode}')
```

Server `error.message` text is for developer logs only. It's English, may be
technical, and is not part of the UX contract.

## 5. Never hide errors

- A failure is shown **inline** (form errors, the failed section of a screen)
  or through **`GlobalSnackBarCubit.showFailure`** (background/global events,
  e.g. a session restore that failed for a reason other than "no session").
- A degraded feature is shown disabled with a reason, not removed silently.
- Expected silence is explicit and commented, e.g. `SessionCubit.restore`
  stays quiet on `UnauthorizedFailure` because "no stored session" is the
  normal first launch.
- After a partial failure, tell the user what still happened. For example,
  logout clears the local session even when server revocation failed, and
  reports the failure.

## 6. Logging

- Use `dart:developer` `log(message, name: 'veemon.<area>', error:, stackTrace:)`.
  **Never `print`** (`avoid_print` is on).
- Log the exception **type and status**, never request/response bodies,
  tokens, emails, phones, or names (see [data-classification](data-classification.md)).
- Crash reporting (when added) scrubs PII in `beforeSend` and is enabled only
  after consent (see [compliance-pdpa](compliance-pdpa.md)).
- Unhandled errors: when a crash reporter is added, wire
  `FlutterError.onError` and `PlatformDispatcher.instance.onError` in
  `main.dart`. Never swallow them.
