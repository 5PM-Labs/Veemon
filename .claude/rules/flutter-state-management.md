---
paths:
  - "apps/mobile/**"
---

# Flutter State Management — Cubit + freezed (`apps/mobile`)

State lives in **Cubits** (`flutter_bloc`). States are **freezed 3 sealed
unions** consumed with Dart 3 pattern matching. Layering is in
[flutter-architecture](flutter-architecture.md); error mapping in
[flutter-error-handling](flutter-error-handling.md).

## 1. Cubits call use cases — nothing else

A cubit's constructor takes **use cases only**: no repositories, data sources,
`Dio`, `ApiClient`, `TokenStorage`, or other cubits.

```dart
// ✓ Good
@injectable
class LoginCubit extends Cubit<LoginState> {
  LoginCubit(this._login) : super(const LoginState.initial());
  final Login _login;
}

// ✗ Bad: skips the domain layer, and couples two cubits
class LoginCubit extends Cubit<LoginState> {
  LoginCubit(this._repository, this._sessionCubit) : super(...);
  final AuthRepository _repository;
  final SessionCubit _sessionCubit;
}
```

- No `BuildContext`, widgets, `Navigator`, `GoRouter`, or l10n strings in a
  cubit. It emits **state**; the UI decides how to render and navigate.
- No business rules in a cubit (validation, permission logic, calculations).
  Those go in the use case.

## 2. Read cubits vs write cubits

| | Read cubit | Write cubit |
|--|-----------|-------------|
| Purpose | Shared, long-lived state (who is signed in, global notices) | One user action on one screen (submit a form) |
| Examples | `SessionCubit`, `GlobalSnackBarCubit` | `LoginCubit` |
| Provided | Once, in `app/app.dart` `MultiBlocProvider` | `BlocProvider(create: (_) => getIt<XCubit>())` in the screen |
| Lifetime | App | Screen, closed automatically by `BlocProvider` |

Never put a write cubit at the app root "so it's easy to reach", and never
create a read cubit inside a screen (you'd get a second, out-of-sync copy).

## 3. Cubits coordinate through the UI

When one cubit's outcome should change another's state, a `BlocListener` in
the widget tree connects them.

```dart
// ✓ Good: features/auth/presentation/screens/login_screen.dart
BlocListener<LoginCubit, LoginState>(
  listenWhen: (_, current) => current is LoginSuccess,
  listener: (context, state) {
    if (state case LoginSuccess(:final user)) {
      context.read<SessionCubit>().signedIn(user);
    }
  },
  child: ...,
)
```

App-wide reactions (session ended → snackbar) are listeners in `app/app.dart`.

## 4. States are sealed freezed unions

```dart
// ✓ Good
@freezed
sealed class LoginState with _$LoginState {
  const factory LoginState.initial() = LoginInitial;
  const factory LoginState.submitting() = LoginSubmitting;
  const factory LoginState.success(UserEntity user) = LoginSuccess;
  const factory LoginState.failure(Failure failure) = LoginFailure;
}

// ✗ Bad: flag soup, so impossible combinations become representable
class LoginState {
  final bool isLoading;
  final UserEntity? user;
  final String? error;   // also: a raw string, not a Failure
}
```

- Union/state → `@freezed sealed class`; single data class →
  `@freezed abstract class`.
- Name every variant (`= LoginSuccess`) so it can be matched by type.
- State carries **domain types** (`UserEntity`, `Failure`), never DTOs or
  localized strings.
- Use the separate `*_state.dart` file next to its cubit.

## 5. Consume with `switch` patterns, not `.when()` / `.map()`

freezed 3 removed `.when()`/`.map()`. Use exhaustive `switch` so a new variant
is a compile error everywhere it isn't handled.

```dart
// ✓ Good: exhaustive switch expression over Either and state unions
emit(switch (result) {
  Right(:final value) => LoginState.success(value),
  Left(:final value) => LoginState.failure(value),
});

final user = context.select<SessionCubit, UserEntity?>(
  (cubit) => switch (cubit.state) {
    SessionAuthenticated(:final user) => user,
    _ => null,
  },
);

// ✓ Good: if-case for a single variant
if (state case SessionUnauthenticated(:final failure?)) { ... }
```

Use a `_ =>` wildcard only when you really mean "every other variant". For
state → UI mapping, list each variant so new ones aren't silently swallowed.

## 6. Async transitions

1. **Guard re-entry:** `if (state is LoginSubmitting) return;`
2. **Emit loading** before the first `await`.
3. **Emit exactly one terminal state** covering every outcome (success *and*
   each failure). There's no path where the cubit stays in `submitting`.

```dart
Future<void> submit({required String email, required String password}) async {
  if (state is LoginSubmitting) return;
  emit(const LoginState.submitting());
  final result = await _login(email: email, password: password);
  emit(switch (result) {
    Right(:final value) => LoginState.success(value),
    Left(:final value) => LoginState.failure(value),
  });
}
```

- Cubits don't `try/catch`. Use cases return `Either`, so there is nothing to
  catch. A thrown exception reaching a cubit is a bug in the data layer.
- If a cubit can be closed mid-request (screen popped), check
  `if (isClosed) return;` before emitting after a long `await`.

## 7. Resources and cleanup

A cubit that opens a `StreamSubscription`, `Timer`, or controller cancels it
in `close()`:

```dart
@override
Future<void> close() async {
  await _subscription?.cancel();
  _debounce?.cancel();
  return super.close();
}
```

## 8. Rebuild narrowly

| Need | Use |
|------|-----|
| One field of the state | `context.select<C, T>(...)` / `BlocSelector` |
| Whole state drives a subtree | `BlocBuilder` with `buildWhen` |
| Side effect (navigate, snackbar, call another cubit) | `BlocListener` (never inside `build`) |
| Both | `BlocConsumer` (rarely needed) |

Put the selector in the **smallest** widget that needs it (see `_SubmitButton`
and `_LoginError` in `login_form.dart`), not at the top of the screen.
