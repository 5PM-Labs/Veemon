# Mobile Security (OWASP MASVS v2) — `apps/mobile`

The Flutter app runs on a device we don't control: assume it is rooted,
instrumented, and its binary decompiled. Security decisions are enforced by the
Go API; the app's job is to **not leak** data and credentials. Organized by the
OWASP Mobile Application Security Verification Standard (MASVS v2) groups.

## MASVS-STORAGE — data at rest

- Tokens and any credential live **only** in `flutter_secure_storage`
  (Android Keystore-backed EncryptedSharedPreferences / iOS Keychain with
  `first_unlock_this_device`). Accessed only through `core/storage/secure_storage.dart`.
- **Never** put tokens or PII in `shared_preferences`, plain files, SQLite
  without encryption, or app logs.
- Clear all secure storage + caches on logout and on 401 after refresh fails.
- Disable Android auto-backup of app data (`android:allowBackup="false"`) so
  the keystore-wrapped data isn't copied off-device.
- Mark sensitive screens (login, profile, payment) to block screenshots /
  app-switcher previews when the product requires it.

## MASVS-CRYPTO

- Don't roll your own crypto; don't embed static keys in the binary. Use the
  platform keystore. Randomness from `Random.secure()`.

## MASVS-AUTH

- Authentication and authorization happen on the server; client-side role
  checks are UX only (hiding a button), never a security boundary.
- Biometric unlock (if added) gates access to a keystore-bound secret
  (`local_auth` + crypto object), not a boolean flag.
- Session timeout: honor server token expiry; refresh via `/api/v1/auth/refresh`;
  on refresh failure, log out.

## MASVS-NETWORK

- HTTPS only in non-debug builds: Android `usesCleartextTraffic="false"` (a
  debug-only network-security-config may allow `10.0.2.2` for the local API),
  iOS ATS left enabled.
- Certificate pinning for production API hosts when the risk profile requires
  it (pin the SPKI of an intermediate/backup key; ship a rotation plan).
- Never disable certificate validation (`badCertificateCallback => true`).

## MASVS-PLATFORM

- Deep links / app links: validate every parameter; never auto-execute actions
  (e.g. "delete account") from a link without user confirmation.
- WebViews: JavaScript off unless needed; no `file://` access; never load
  untrusted URLs with a JS bridge attached.
- Only request permissions the feature needs, at the moment it's needed.
- Don't put PII in notifications' lock-screen content or in the clipboard.

## MASVS-CODE

- Base URL and non-secret config come from `--dart-define` (see
  `lib/core/config/app_config.dart`). **No secrets** in `--dart-define` either —
  they are extractable from the binary.
- Release builds: `--obfuscate --split-debug-info`. Keep `minSdkVersion`/iOS
  deployment target on supported, patched OS versions.
- Dependencies scanned (see [supply-chain-security](supply-chain-security.md)).
- Validate/sanitize server data before rendering (it's still untrusted).

## MASVS-RESILIENCE (risk-based)

For high-value apps (finance, health): root/jailbreak detection, Play Integrity /
App Attest sent to the API for sensitive flows, anti-tamper. These raise cost
for attackers; they don't replace server-side checks.

## MASVS-PRIVACY

- Follows [compliance-pdpa](compliance-pdpa.md): consent before analytics/crash
  reporting that includes identifiers; no PII in analytics events.
- App Store privacy labels / Play Data safety form updated when data collection
  changes.
- In-app account deletion (required by both stores) wired to the server erasure
  flow.
