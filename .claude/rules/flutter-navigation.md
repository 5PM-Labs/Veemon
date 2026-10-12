---
paths:
  - "apps/mobile/**"
---

# Flutter Navigation — go_router (`apps/mobile`)

Routing lives in `lib/app/router.dart` (`createRouter`, `AppRoutes`).

## 1. Paths are constants

```dart
// ✓ Good
context.go(AppRoutes.home);

// ✗ Bad: a typo becomes a runtime 404
context.go('/hom');
```

Every route's path is declared in `AppRoutes`. Parameterized routes get a
builder: `static String userDetail(String id) => '/users/${Uri.encodeComponent(id)}';`.

## 2. Auth-dependent navigation is a redirect, not a push

The router's `redirect` is a **pure function of `SessionCubit` state**, and
`refreshListenable` re-runs it on every session change.

```dart
// ✓ Good: login screen only reports success. The redirect moves the user.
context.read<SessionCubit>().signedIn(user);

// ✗ Bad: screens deciding auth flow
context.go(AppRoutes.home);   // right after login
Navigator.of(context).pushReplacement(...);
```

- Protected routes need no per-screen checks. An unauthenticated session can
  never render them.
- Return `null` from `redirect` when already at the target (avoids loops).
- Never `await` I/O inside `redirect`. Read the cubit's current state only.

## 3. `go` vs `push`

| | Use for |
|--|--------|
| `context.go(path)` | Switching to a location (tabs, top-level sections). Replaces the stack. |
| `context.push(path)` | Drill-down where Back should return (list → detail). |
| `context.pop(result)` | Returning a value from a pushed route/dialog. |

Don't mix `Navigator.push(MaterialPageRoute(...))` with go_router for app
screens. Dialogs and bottom sheets (`showDialog`, `showModalBottomSheet`) are
fine.

## 4. Route parameters are untrusted input

Path and query parameters can come from deep links, i.e. from anyone.

```dart
GoRoute(
  path: '/users/:id',
  builder: (_, state) {
    final id = state.pathParameters['id'];
    if (id == null || !_uuid.hasMatch(id)) return const NotFoundScreen();
    return UserDetailScreen(userId: id);
  },
),
```

- Validate the format before use, and pass **IDs** to the screen (not whole
  objects via `extra`). The screen loads data through its cubit, so the server
  still authorizes access.
- A deep link never performs a destructive or state-changing action without
  explicit user confirmation (see [mobile-security](mobile-security.md)
  MASVS-PLATFORM).
- Never put tokens or PII in paths or query strings. They end up in logs and
  analytics.

## 5. Where providers go

- Global read cubits are above `MaterialApp.router` (`app/app.dart`), so
  every route can `context.read` them.
- A write cubit is created in its screen's `BlocProvider`, not in the
  `GoRoute.builder`. The screen owns its dependencies.
- Splash (`AppRoutes.splash`) is shown only while `SessionState.unknown`.
  Don't add artificial delays.
