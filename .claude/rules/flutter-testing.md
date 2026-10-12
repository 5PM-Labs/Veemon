# Flutter Testing (`apps/mobile`)

## What to test (priority order)

1. **Use cases** — business rules, with mocked repository interfaces.
2. **Cubits** — state transitions with `blocTest` and mocked use cases.
3. **Repository impls** — exception → `Failure` mapping and side effects
   (token stored / cleared), with mocked data sources.
4. **Core infra** — `ApiClient` envelope parsing (fake `HttpClientAdapter`),
   `AppConfig` validation, validators, DI graph resolves.

Skip low-value tests of trivial DTO getters and pixel-level widget trees.

## Conventions

- Mocks via mockito `@GenerateNiceMocks([MockSpec<T>()])` in
  `test/features/<feature>/helpers/<feature>_mocks.dart`; run `make gen`.
  Mock **interfaces / use cases**, never the class under test.
- Generic return types (`Future<Either<Failure, T>>`) need
  `provideDummy` — call `registerDummies()` from `test/helpers/test_helper.dart`
  in `setUpAll`.
- Table-driven cases with Dart records:
  ```dart
  final cases = <(String, AppException, Failure)>[
    ('401 → invalid credentials', const ApiException(statusCode: 401), const Failure.invalidCredentials()),
    ('offline → network', const NetworkException(), const Failure.network()),
  ];
  for (final (name, exception, failure) in cases) {
    test(name, () async { ... });
  }
  ```
- Cover success **and** each failure path; `verify` side effects and
  `verifyZeroInteractions` when validation must short-circuit.
- No `Future.delayed` / real timers — use `fake_async` (`async.elapse`).
- **Synthetic data only** (`@example.test` emails, fake IDs) — never real
  personal data in fixtures ([compliance-pdpa](compliance-pdpa.md)).
- One test file per source file, same path under `test/`, `_test.dart` suffix.

## Run

```bash
cd apps/mobile
make test            # flutter test
make test-coverage   # + coverage/lcov.info
make check           # gen + format-check + analyze + test (what CI runs)
```
