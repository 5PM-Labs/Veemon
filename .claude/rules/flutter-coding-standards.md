---
paths:
  - "apps/mobile/**"
---

# Flutter Coding Standards — Dart code quality & workflow (`apps/mobile`)

The topic rules for the mobile app:

| Topic | Rule |
|-------|------|
| Layers, DI, data layer, file structure | [flutter-architecture](flutter-architecture.md) |
| Cubits, freezed states, pattern matching | [flutter-state-management](flutter-state-management.md) |
| Widgets, theme, l10n, a11y, performance | [flutter-ui](flutter-ui.md) |
| `Either`, `Failure`, `guard()`, user messages | [flutter-error-handling](flutter-error-handling.md) |
| go_router, redirects, deep links | [flutter-navigation](flutter-navigation.md) |
| Config, builds, signing, store compliance | [flutter-release](flutter-release.md) |
| Tests | [flutter-testing](flutter-testing.md) |
| Security (MASVS) | [mobile-security](mobile-security.md) |

This file covers what's left: Dart style and the day-to-day workflow.

## 1. Analyzer: zero issues

- `flutter analyze` must report **zero** issues (CI enforces it).
  `analysis_options.yaml` enables `strict-casts`, `strict-raw-types`, and
  extra lints (`prefer_final_locals`, `unawaited_futures`, `cancel_subscriptions`,
  `close_sinks`, `avoid_print`, …).
- No `// ignore:` / `// ignore_for_file:` without a comment saying **why**.
  Never add one just to make CI green.
- `dart format` clean. CI runs `--set-exit-if-changed`. Use trailing commas
  to get readable multi-line formatting.

## 2. Naming & files

| Thing | Convention | Example |
|-------|-----------|---------|
| Files, dirs | `snake_case`, named after the one public class | `login_cubit.dart` |
| Classes, enums, typedefs, extensions | `UpperCamelCase` | `AuthRepositoryImpl`, `FailureMessageX` |
| Members, variables, params | `lowerCamelCase` | `tokenStorage` |
| Constants | `lowerCamelCase` (Dart style, **not** `SCREAMING_CAPS`) | `const defaultTimeout = ...` |
| Private | leading `_` | `_SubmitButton`, `_tokenStorage` |
| Interface + impl | `XRepository` / `XRepositoryImpl`, `XDataSource` / `XDataSourceImpl` | |
| Use case | verb phrase, callable via `call()` | `Login`, `GetCurrentUser` |
| Cubit / state | `XCubit` + `XState`, variants `XSuccess`, `XFailure`… | `LoginCubit`, `LoginSubmitting` |
| Extension | `<Type>X` or a purpose name | `L10nX`, `UserDtoMapper` |

- One public class per file. Small private helpers (`_PrivateWidget`) may
  live with their owner.
- Keep files under ~300 lines. Split widgets or move logic into a use case
  when they grow.

## 3. Imports

- **Relative** within the same feature (`'../cubits/login_cubit.dart'`).
- **`package:veemon_mobile/...`** for `core/` and other features.
- Never import another feature's `data/` or `presentation/`. Go through its
  `domain/`.
- No unused imports. Order: `dart:` → `package:` → relative (the formatter
  and analyzer handle the rest).

## 4. Dart idioms

- Make everything that can be `final` final. Prefer immutable classes
  (freezed / `const` constructors).
- Use `abstract interface class` for contracts (`AuthRepository`,
  `TokenStorage`), `abstract final class` for static-only namespaces
  (`ApiEndpoints`, `AppTheme`, `EmailValidator`), and `sealed class` for
  closed hierarchies (`AppException`).
- Use **records and patterns** instead of ad-hoc tuples/maps:
  `final (name, input, expected) = ...`, `if (x case Foo(:final bar))`.
- Use null-safety deliberately: no `!` on values that can legitimately be
  null. Use `?.`, `??`, or a pattern. `late` only when initialization is
  guaranteed (e.g. `late final _router = createRouter(...)`).
- Mark futures you intentionally don't await with `unawaited(...)`. Never
  just drop them.
- Don't use `dynamic` except at the JSON boundary (`Map<String, dynamic>` in
  DTO `fromJson`).
- Document **why**, not what. Doc comments (`///`) on public APIs whose
  behavior isn't obvious from the name (see `TokenStorage`, `guard`).

## 5. Generated code

- After touching any `@freezed` / `@JsonSerializable` / `@injectable` class,
  mock spec, or `.arb` file, run **`make gen`** (or `make watch` while
  iterating).
- Never edit generated files (`*.g.dart`, `*.freezed.dart`, `*.config.dart`,
  `*.mocks.dart`, `lib/core/language/gen/`). They're git-ignored and
  regenerated in CI.
- A `part '...'` directive must match the file name exactly.

## 6. Dependencies

- Add a package only when it clearly pays for itself. Prefer Flutter
  Favorites / verified publishers with recent releases. State the reason in
  the PR (see [supply-chain-security](supply-chain-security.md)).
- Use `flutter pub add` / `flutter pub add --dev`, and commit `pubspec.lock`.
- No `dependency_overrides` or git dependencies without a comment and an
  owner.

## 7. Workflow

```bash
cd apps/mobile
make gen      # after model / DI / l10n changes
make check    # gen + format-check + analyze + test — run before every push
make build-apk / make build-ios   # after native config changes
```

- Conventional commits with the `mobile` scope: `feat(mobile): add register screen`.
- Track work in issues, not `TODO` comments. A `TODO` needs an issue link.
- Before opening a PR, run the [mobile-reviewer](../agents/mobile-reviewer.md)
  agent.
