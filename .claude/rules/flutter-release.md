---
paths:
  - "apps/mobile/**"
---

# Flutter Config, Build & Release (`apps/mobile`)

## 1. Configuration

- The only build-time config is **non-secret** `--dart-define` values, read
  once in `core/config/app_config.dart` (`String.fromEnvironment`) and
  injected as the `AppConfig` singleton. Never call `String.fromEnvironment`
  anywhere else.
- `AppConfig` **fails fast**, like Go's `Config.Validate()`: a release build
  with a non-`https` `API_BASE_URL` throws at startup. Validate every new key
  there too.
- **No secrets** in `--dart-define`, assets, or source. Anything in the binary
  is public. Server-side secrets stay behind the Go API (see
  [supply-chain-security](supply-chain-security.md)).
- New key: add it to `AppConfig`, the `Makefile` (`run`/`build-*` targets),
  `apps/mobile/README.md`, and the CI build step.
- Per-environment builds (dev/staging/prod) use
  `--dart-define-from-file=env/<env>.json` (non-secret values only, committed)
  before reaching for native flavors. Add flavors only when the app ID or
  icon must differ.

## 2. Versioning

- `pubspec.yaml` `version: <semver>+<build>`. Bump semver for user-visible
  changes, and the build number **every** store upload (it must increase
  monotonically). CI may override it with `--build-number`.
- The bundle ID / applicationId (`com.veemon.veemon_mobile`, a placeholder)
  is permanent after the first store upload. Decide it before then.

## 3. Builds

| Purpose | Command |
|---------|---------|
| Compile check after native changes | `make build-apk` / `make build-ios` |
| Android release | `flutter build appbundle --release --obfuscate --split-debug-info=build/symbols --dart-define=API_BASE_URL=https://...` |
| iOS release | `flutter build ipa --release --obfuscate --split-debug-info=build/symbols --dart-define=API_BASE_URL=https://...` |

- Release builds are produced **by CI from a tagged commit**, never from a
  laptop.
- Keep `build/symbols` as a CI artifact and upload it to the crash reporter.
  Don't ship it in the app.
- Signing (keystore, `key.properties`, certificates, provisioning profiles)
  comes from CI secrets. It is git-ignored and never committed.

## 4. Native configuration changes

Changes to `android/` or `ios/` (manifest, Gradle, `Info.plist`,
entitlements) need:

- A compile check (`make build-apk` / `make build-ios`) in the PR.
- No weakening of the hardening: `allowBackup="false"`,
  `usesCleartextTraffic="false"` in the main manifest (cleartext only in
  `src/debug`), ATS enabled with only `NSAllowsLocalNetworking`.
- Permissions requested only for a shipped feature, with a usage-description
  string (`NS*UsageDescription`) that explains why in plain language.
- Keep `minSdk` / iOS deployment target on OS versions that still get
  security patches.

## 5. Store compliance

- **Account deletion** must be available in-app (Apple and Google
  requirement). Wire it to the server erasure flow (see
  [compliance-pdpa](compliance-pdpa.md) §3).
- Update the App Store privacy "nutrition label" and the Play **Data safety**
  form in the same release that changes data collection (a new SDK, a new
  field, analytics).
- Third-party SDKs (analytics, crash, ads) are reviewed like any vendor:
  data classes they receive, region, DPA (see
  [compliance-soc2](compliance-soc2.md) CC9). Add them to the iOS privacy
  manifest (`PrivacyInfo.xcprivacy`) if they require one.

## 6. Dependency upgrades

- `flutter pub outdated` → upgrade → `make check` → a compile check on both
  platforms.
- Commit `pubspec.lock` (this is an app, not a package).
- Pin the Flutter version in CI (`FLUTTER_VERSION` in `.github/workflows/ci.yml`)
  and bump it deliberately, together with `environment.sdk` in `pubspec.yaml`.
