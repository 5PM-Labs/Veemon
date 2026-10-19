---
paths:
  - "apps/mobile/**"
---

# Flutter Architecture (`apps/mobile`)

Feature-first **Clean Architecture** with Cubits, adapted from
[adamsmaka's Flutter rules](https://gist.github.com/adamsmaka/ce70c883ad9ad8cb6256fd10fbbf8b71)
to this monorepo: the backend is the Go REST API (not Firestore), errors use
`fpdart`'s `Either` (maintained successor of `dartz`, same API), and the gist's
premium/RevenueCat section doesn't apply. Paths are relative to `apps/mobile/`.

## 1. Layer flow — one direction only

```
Widget → Cubit → UseCase → Repository (interface) ← RepositoryImpl → DataSource → ApiClient → Go API
         presentation      domain                   data
```

- **domain/** is pure Dart: entities, repository *interfaces*, use cases. No
  `package:flutter`, `dio`, `json_annotation`, or `get_it` imports.
- **data/** implements domain interfaces: DTOs, mappers, data sources,
  `*RepositoryImpl`. Depends on domain, never on presentation.
- **presentation/** depends on domain only (entities, use cases, `Failure`) —
  never on DTOs, data sources, or `ApiClient`.

```dart
// ✓ Good: business rule in the use case
class Login {
  Future<Either<Failure, UserEntity>> call({required String email, required String password}) async {
    if (!EmailValidator.isValid(email)) {
      return const Left(Failure.validation(ValidationIssue.invalidEmail));
    }
    return _repository.login(email: email.trim(), password: password);
  }
}

// ✗ Bad: validation + HTTP inside a cubit
class LoginCubit extends Cubit<LoginState> {
  Future<void> submit(String email, String password) async {
    if (!email.contains('@')) { ... }
    final res = await dio.post('/api/v1/auth/login', ...);
  }
}
```

## 2. No cross-dependencies

- A repository never calls another repository; a cubit never injects another
  cubit. Work spanning domains → a use case that takes both repositories.
- Cubits coordinate through the UI with `BlocListener` (e.g. `LoginScreen`
  listens for `LoginSuccess` and calls `SessionCubit.signedIn`).

## 3. Read vs write cubits

- **Read cubits** hold shared state and are provided once at the app root
  (`SessionCubit`, `GlobalSnackBarCubit` in `app/app.dart`).
- **Write cubits** are scoped to the screen that performs the action
  (`LoginCubit` via `BlocProvider` in `LoginScreen`) and die with it.
- Navigation that depends on auth is a function of `SessionCubit` state in the
  router's `redirect` — screens don't push login/home themselves.

Details: [flutter-state-management](flutter-state-management.md),
[flutter-navigation](flutter-navigation.md).

## 4. Dependency injection — `injectable` + `get_it`

- Annotations only; never call `getIt.registerX` by hand.
  | Kind | Annotation |
  |------|-----------|
  | Repository impl, data source | `@LazySingleton(as: Interface)` |
  | Use case, cubit | `@injectable` (factory) |
  | Third-party types (`Dio`, `FlutterSecureStorage`, `AppConfig`) | `@module` in `core/di/injectable_module.dart` |
- Depend on interfaces (`AuthRepository`, `TokenStorage`), inject via the
  constructor. `getIt<T>()` is called only in `BlocProvider(create:)` and
  `main.dart` — never inside domain/data classes.

## 5. Data layer

- **DTOs mirror the proto wire shape** (`contract/<svc>/<svc>.proto`,
  protojson camelCase). The server marshals with `EmitUnpopulated: true`
  (`internal/pkg/response`), so empty strings/zeros are sent and unset messages arrive
  as `null`; still give scalars `@Default(...)` so parsing stays tolerant.
  **`int64`/`uint64` arrive as JSON strings** — type them `String` and parse in
  the mapper (see the `flutter-api-endpoint` skill).
- **`ApiClient` is the only Dio user.** It unwraps the `internal/pkg/response` envelope
  (`{success, data, error:{code,message}}`) and throws `AppException`s.
- **Mapping via extensions** (`dto.toEntity()`), in `data/mappers/`.
- **Repositories return `Either<Failure, T>`** and use `guard()` to convert
  exceptions; endpoint-specific statuses go in `onApiException`
  (401 on `/login` = invalid credentials, not an expired session).
- Paths live in `core/network/api_endpoints.dart` and must match the proto
  `veemon.route` options. A contract change updates DTOs + endpoints in the
  same PR.
- Server-managed fields (`id`, `createdAt`) are never sent in request bodies.
- Streams/controllers a repository exposes are broadcast and closed in
  `@disposeMethod`.

## 6. File structure

```
lib/
├── main.dart
├── app/                     app widget, router
├── core/                    shared only — no feature code, no "misc utils" dumping ground
│   ├── config/ di/ error/ network/ storage/ snackbar/ language/ theme/ widgets/ utils/
└── features/<feature>/
    ├── domain/{entities,repositories,use_cases}/
    ├── data/{dtos,mappers,data_sources,repositories}/
    └── presentation/{cubits,screens,widgets}/
test/                        mirrors lib/ exactly; shared helpers in test/helpers/,
                             feature helpers in test/features/<feature>/helpers/
```

- One public class per file, file named after it in `snake_case`
  (`login_cubit.dart` → `LoginCubit`); a cubit's state gets its own
  `*_state.dart`.
- **Imports:** relative within a feature; `package:veemon_mobile/...` for
  `core/` and other features. A feature importing another feature's `data/`
  is a layering bug — go through its domain.
