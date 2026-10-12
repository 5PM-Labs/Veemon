import 'package:flutter_test/flutter_test.dart';
import 'package:veemon_mobile/core/config/app_config.dart';

void main() {
  group('AppConfig', () {
    test('accepts http in debug builds (local API)', () {
      expect(
        AppConfig(
          apiBaseUrl: 'http://10.0.2.2:3000',
          isRelease: false,
        ).apiBaseUrl,
        'http://10.0.2.2:3000',
      );
    });

    test('rejects http in release builds', () {
      expect(
        () => AppConfig(apiBaseUrl: 'http://api.example.com', isRelease: true),
        throwsArgumentError,
      );
    });

    test('accepts https in release builds', () {
      expect(
        AppConfig(
          apiBaseUrl: 'https://api.example.com',
          isRelease: true,
        ).apiBaseUrl,
        'https://api.example.com',
      );
    });

    test('rejects a non-URL', () {
      expect(
        () => AppConfig(apiBaseUrl: 'not a url', isRelease: false),
        throwsArgumentError,
      );
    });
  });
}
