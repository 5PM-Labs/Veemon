---
name: flutter-feature
description: Scaffold a new feature (or a new screen/use case inside one) in the Flutter app apps/mobile following its Clean Architecture — domain entities/repository/use cases, data DTOs/data source/repository impl, presentation cubit/state/screen, DI, routes, l10n, and tests. Use whenever a task adds a feature, screen, or user flow to apps/mobile.
paths:
  - "apps/mobile/**"
---

# Add a feature to `apps/mobile`

Follow the rules in `.claude/rules/flutter-*.md`. The `auth` feature
(`lib/features/auth/`) is the reference implementation; mirror its shape.
Below, `<f>` is the feature (`snake_case`, e.g. `orders`), and `<Thing>` is
the entity (`Order`).

## 0. Decide the slice

- Which **use cases** (verbs: `GetOrders`, `CancelOrder`)? Each one is a
  class with `call()`.
- Which **read state** is shared (a global read cubit at the app root), and
  which actions are screen-scoped (write cubits)?
- Which **API routes**? If they don't exist in `contract/*.proto` yet, use the
  `proto-routes` skill for the Go side first, then `flutter-api-endpoint`.
- Does it touch **PII**? Classify fields (see `data-classification.md`) and
  keep them out of logs.

## 1. Domain (pure Dart, no Flutter/Dio/JSON imports)

```
lib/features/<f>/domain/
├── entities/<thing>_entity.dart          @freezed abstract class <Thing>Entity
├── repositories/<f>_repository.dart      abstract interface class <F>Repository
└── use_cases/<verb>_<thing>.dart         @injectable class <Verb><Thing>
```

```dart
abstract interface class OrderRepository {
  Future<Either<Failure, List<OrderEntity>>> getOrders({int page = 1});
}

@injectable
class GetOrders {
  GetOrders(this._repository);
  final OrderRepository _repository;

  Future<Either<Failure, List<OrderEntity>>> call({int page = 1}) =>
      _repository.getOrders(page: page);
}
```

Business rules and input validation go in the use case. Return
`Left(Failure.validation(...))` and add a `ValidationIssue` value plus an ARB
message if needed.

## 2. Data

```
lib/features/<f>/data/
├── dtos/<thing>_dto.dart                     @freezed + fromJson, mirrors the proto message
├── mappers/<thing>_mapper.dart               extension <Thing>DtoMapper on <Thing>Dto { toEntity() }
├── data_sources/<f>_remote_data_source.dart  interface + @LazySingleton(as:) impl using ApiClient
└── repositories/<f>_repository_impl.dart     @LazySingleton(as: <F>Repository), wraps calls in guard()
```

See the `flutter-api-endpoint` skill for DTO and endpoint details.

## 3. Presentation

```
lib/features/<f>/presentation/
├── cubits/<name>_cubit.dart + <name>_state.dart   @injectable, freezed sealed state
├── screens/<name>_screen.dart                     BlocProvider(create: (_) => getIt<XCubit>()) + listeners
└── widgets/<name>.dart                            content widgets, private _Widget classes
```

- The cubit takes **use cases only**. Loading → one terminal state, with a
  re-entry guard.
- The screen owns its write cubit. A global read cubit goes in `app/app.dart`'s
  `MultiBlocProvider`.
- UI text goes in **both** `app_en.arb` and `app_id.arb`. Colors come from
  `colorScheme`.

## 4. Wire it up

- **Route:** add a constant to `AppRoutes` and a `GoRoute` in
  `lib/app/router.dart`. Validate path params, and let the redirect handle
  auth.
- **DI:** annotations only, then run `make gen`. Add new cubits to
  `test/core/di/service_locator_test.dart`.
- **Errors:** a new `Failure` subtype? Update `failures.dart`,
  `failure_message.dart`, and both ARB files.

## 5. Tests (mirror `lib/` under `test/`)

```
test/features/<f>/
├── helpers/<f>_mocks.dart        @GenerateNiceMocks([MockSpec<<F>Repository>(), MockSpec<<F>RemoteDataSource>(), MockSpec<<Verb><Thing>>()])
├── helpers/<f>_test_data.dart    synthetic fixtures only
├── domain/use_cases/<verb>_<thing>_test.dart
├── data/repositories/<f>_repository_impl_test.dart
└── presentation/cubits/<name>_cubit_test.dart
```

If a use case returns a new `Either<Failure, X>` type, register a dummy in
`test/helpers/test_helper.dart` → `registerDummies()`.

## 6. Verify

```bash
make -C apps/mobile gen
make -C apps/mobile check        # format + analyze (0 issues) + tests
make -C apps/mobile build-apk    # only if android/ or ios/ changed
```

Then run the `mobile-reviewer` agent. In the PR, tick `apps/mobile` and fill
in the *Security & compliance* section.
