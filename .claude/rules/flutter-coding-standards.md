# Flutter Coding Standards (`apps/mobile`)

State management, UI, error handling and workflow rules. Architecture lives in
[flutter-architecture](flutter-architecture.md); tests in
[flutter-testing](flutter-testing.md); security in
[mobile-security](mobile-security.md).

## State (Cubit + freezed 3)

- States and unions are `@freezed sealed class`; plain data classes are
  `@freezed abstract class`.
- Consume with Dart 3 **`switch` expressions / patterns** — not freezed's
  `.when()` / `.map()` (removed in freezed 3).
  ```dart
  // ✓ Good
  emit(switch (result) {
    Right(:final value) => LoginState.success(value),
    Left(:final value) => LoginState.failure(value),
  });
  ```
- Emit a loading state before async work, then exactly one terminal state;
  guard against double submission (`if (state is LoginSubmitting) return;`).
- Cancel subscriptions/timers in `close()`.
- Rebuild narrowly: `context.select` / `BlocSelector` for one field;
  `BlocListener` for side effects (navigation, snackbars), never `BlocBuilder`.

## UI

- **No user-facing string literals** — everything goes through `context.l10n`
  (ARB files in `lib/core/language/l10n/`, `en` is the template, `id` must stay
  in sync).
- **Theme only:** `Theme.of(context).colorScheme` / `textTheme`. No
  `Colors.red`, no hex literals outside `core/theme/`.
- **Widgets, not helper methods:** extract `_PrivateWidget` classes instead of
  `Widget _buildHeader()` — they get their own `const` constructor and
  rebuild boundary.
- `const` everywhere it compiles (`prefer_const_constructors` is on).
- After every `await` in a widget: `if (!context.mounted) return;` before
  using `context` (also in `catch`).
- Responsive decisions from `LayoutBuilder` constraints; `MediaQuery` only for
  device facts (padding, text scale).
- Sensitive inputs: `obscureText`, `enableSuggestions: false`,
  `autocorrect: false`; correct `autofillHints` so password managers work.

## Error handling

- Everything below the UI returns **`Either<Failure, T>`**; business logic
  doesn't throw. Data-layer `AppException`s are caught in repositories.
- **Specific failures** (`core/error/failures.dart`): `network`, `timeout`,
  `unauthorized`, `invalidCredentials`, `rateLimited`, `validation(issue)`,
  `server(statusCode, code)`, `unexpected`. Add a subtype when the UI must
  react differently — not a free-text message.
- **Users see localized, actionable messages** (`failure.message(context)`);
  status codes and server messages stay in developer logs.
- **Never hide errors:** a failure is shown inline or via
  `GlobalSnackBarCubit`, or the feature is shown disabled with a reason.
- Logging: `dart:developer` `log()` with a `name:`; never `print`. Never log
  tokens, request/response bodies, or PII ([data-classification](data-classification.md)).

## Workflow

- `flutter analyze` reports **zero** issues; don't add `// ignore:` without
  a comment explaining why.
- `dart format` clean (CI runs `--set-exit-if-changed`).
- After touching any `@freezed` / `@JsonSerializable` / `@injectable` class,
  mock spec, or `.arb` file: `make gen`. Never edit generated files; they're
  git-ignored.
- Verify compilation with `make build-apk` / `make build-ios` when you change
  native config (manifests, Gradle, Info.plist).
- Conventional commits: `feat(mobile): ...`, `fix(mobile): ...`.
- Track work in issues, not `TODO` comments.
