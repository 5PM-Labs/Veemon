# @veemon/mobile — Flutter client

iOS + Android client for the Veemon Go API (`apps/api`). Feature-first **Clean
Architecture** with Cubit state management. The rules it follows live in
[`.claude/rules/flutter-*.md`](../../.claude/rules/) (start with
[`flutter-coding-standards.md`](../../.claude/rules/flutter-coding-standards.md),
which indexes the rest) and
[`mobile-security.md`](../../.claude/rules/mobile-security.md). To add a
feature or a backend call, follow the `flutter-feature` and
`flutter-api-endpoint` skills in [`.claude/skills/`](../../.claude/skills/).

Not a Bun workspace member. Orchestrated with `make` like `apps/api`.

## Quickstart

```bash
# Prerequisites: Flutter 3.47+ (Dart 3.13+); Android SDK and/or Xcode for device builds
cd apps/mobile
make gen        # pub get + gen-l10n + build_runner (freezed / json / injectable / mocks)
make check      # format-check + analyze (zero issues) + test — same as CI
make run        # runs against http://10.0.2.2:3000 (the host's Go API from the Android emulator)

# iOS simulator / a real backend:
make run API_BASE_URL=http://localhost:3000
make run API_BASE_URL=https://api.staging.example.com
```

Generated files (`*.g.dart`, `*.freezed.dart`, `*.config.dart`, `*.mocks.dart`,
`lib/core/language/gen/`) are **git-ignored** — run `make gen` after cloning
and after changing any annotated class or `.arb` file.

`API_BASE_URL` is the only build-time config. Release builds refuse a
non-`https` URL at startup (`AppConfig`). Never pass secrets via
`--dart-define` — they're extractable from the binary.

## Layout

```
lib/
├── main.dart                 → configureDependencies() + runApp
├── app/                      → App widget (global providers, listeners), go_router
├── core/                     → shared across features only
│   ├── config/               → AppConfig (dart-define, fail-fast validation)
│   ├── di/                   → get_it + injectable (service_locator, @module)
│   ├── error/                → AppException (data), Failure (domain), guard(), l10n messages
│   ├── network/              → ApiClient (unwraps pkg/response envelope), endpoints, auth interceptor
│   ├── storage/              → TokenStorage → flutter_secure_storage (Keychain / Keystore)
│   ├── snackbar/             → GlobalSnackBarCubit
│   ├── language/             → ARB files (en, id) + context.l10n
│   ├── theme/ widgets/ utils/
└── features/
    └── auth/
        ├── domain/           → entities, repository interface, use cases (pure Dart)
        ├── data/             → DTOs (proto wire shape), mappers, remote data source, repository impl
        └── presentation/     → cubits (SessionCubit global, LoginCubit scoped), screens, widgets
test/                         → mirrors lib/
```

Flow: **Widget → Cubit → UseCase → Repository → DataSource → ApiClient → Go API**.
Every operation below the UI returns `Either<Failure, T>`; the data layer's
exceptions never escape a repository.

## API contract

DTOs mirror the protojson shapes in [`contract/user/user.proto`](../../contract/user/user.proto)
(camelCase) and the `pkg/response` envelope (`{success, data, error:{code,message}}`).
Paths live in `core/network/api_endpoints.dart` — keep them in sync with the
proto `veemon.route` options.

Implemented: login, restore session (`GET /api/v1/auth/me`), logout (server
revoke + local wipe). Not yet: register screen, token refresh
(`/api/v1/auth/refresh`) — on a 401 the app currently signs the user out.

## Security notes

- Token only in `flutter_secure_storage` (`first_unlock_this_device`); wiped on
  logout and when the server rejects it.
- Android: `allowBackup=false`, data-extraction rules exclude everything,
  cleartext HTTP allowed **only** in the debug manifest. iOS: ATS on, only
  `NSAllowsLocalNetworking` for the simulator.
- No HTTP logging interceptor — bodies carry tokens and PII.
- Release: `flutter build appbundle --release --obfuscate --split-debug-info=build/symbols`;
  signing material comes from CI secrets (git-ignored locally).
- Bundle ID is the placeholder `com.veemon.veemon_mobile` — change it before
  the first store upload (it's permanent after that).
