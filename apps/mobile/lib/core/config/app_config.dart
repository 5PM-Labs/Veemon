import 'package:flutter/foundation.dart';

/// Non-secret runtime configuration, injected at build time with
/// `--dart-define=API_BASE_URL=https://api.example.com`.
///
/// Never put secrets here: `--dart-define` values are extractable from the
/// binary (.claude/rules/mobile-security.md, MASVS-CODE).
class AppConfig {
  AppConfig({required this.apiBaseUrl, this.isRelease = kReleaseMode}) {
    _validate();
  }

  /// Reads `API_BASE_URL`; defaults to the local Go API as seen from the
  /// Android emulator (`10.0.2.2` is the host machine).
  factory AppConfig.fromEnvironment() => AppConfig(
    apiBaseUrl: const String.fromEnvironment(
      'API_BASE_URL',
      defaultValue: 'http://10.0.2.2:3000',
    ),
  );

  final String apiBaseUrl;
  final bool isRelease;

  /// Fail fast, like the Go `Config.Validate()`: a release build must talk
  /// to the API over HTTPS (MASVS-NETWORK).
  void _validate() {
    final uri = Uri.tryParse(apiBaseUrl);
    if (uri == null || !uri.hasScheme || uri.host.isEmpty) {
      throw ArgumentError.value(apiBaseUrl, 'API_BASE_URL', 'is not a URL');
    }
    if (isRelease && uri.scheme != 'https') {
      throw ArgumentError.value(
        apiBaseUrl,
        'API_BASE_URL',
        'must use https in release builds',
      );
    }
  }
}
