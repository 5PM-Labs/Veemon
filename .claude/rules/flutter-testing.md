---
paths:
  - "apps/mobile/**"
---

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
- No `Future.delayed` / real timers — use `fake_async` (`async.elapse`;
  add it with `flutter pub add --dev fake_async` the first time it's needed).
- **Synthetic data only** (`@example.test` emails, fake IDs) — never real
  personal data in fixtures ([compliance-pdpa](compliance-pdpa.md)).
- One test file per source file, same path under `test/`, `_test.dart` suffix.

## Widget tests — only where they pay off

Worth it: a widget with real logic (conditional rendering per state, form
wiring, a `BlocListener` that triggers navigation or another cubit). Not worth
it: static layouts or pixel details.

```dart
testWidgets('shows the localized error for invalid credentials', (tester) async {
  final cubit = MockLoginCubit();
  whenListen(cubit, Stream<LoginState>.empty(),
      initialState: const LoginState.failure(Failure.invalidCredentials()));

  await tester.pumpWidget(
    MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: BlocProvider<LoginCubit>.value(value: cubit, child: const Scaffold(body: LoginForm())),
    ),
  );

  expect(find.text('Incorrect email or password.'), findsOneWidget);
});
```

- **Drive state** with `bloc_test`'s `MockCubit` (`class MockLoginCubit
  extends MockCubit<LoginState> implements LoginCubit {}`) plus `whenListen`.
- **Assert method calls** with a small recording fake
  (`class _RecordingLoginCubit extends Cubit<LoginState> implements LoginCubit`).
  `MockCubit` is mocktail-based, and we standardize on mockito, so don't add
  mocktail just for `verify`. Reference:
  `test/features/auth/presentation/widgets/login_form_test.dart`.
- Never hit DI or the network from a widget test.
- Always pump with the real `AppLocalizations` delegates, so the test also
  catches missing ARB keys.
- Find by text/semantics/`Key`, not by widget type position.

## Must-have tests per change

| Change | Test |
|--------|------|
| New use case | success + every failure/validation branch |
| New cubit / new state variant | `blocTest` for each transition |
| New repository method | exception → `Failure` mapping + side effects |
| New `Failure` subtype | `failure_message` covered (exhaustive switch compiles) + repository mapping |
| New DI registration | still resolved by `test/core/di/service_locator_test.dart` (add the cubit there) |
| New config key | `AppConfig` validation test |

## Run

```bash
cd apps/mobile
make test            # flutter test
make test-coverage   # + coverage/lcov.info
make check           # gen + format-check + analyze + test (what CI runs)
```
