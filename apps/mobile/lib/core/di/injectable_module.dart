import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:injectable/injectable.dart';

import '../config/app_config.dart';
import '../network/auth_interceptor.dart';

/// Third-party types that can't carry injectable annotations themselves.
@module
abstract class InjectableModule {
  @singleton
  AppConfig get appConfig => AppConfig.fromEnvironment();

  /// Keychain / Keystore-backed storage. `first_unlock_this_device` keeps the
  /// item out of iCloud backups and off other devices (MASVS-STORAGE).
  @lazySingleton
  FlutterSecureStorage get secureStorage => const FlutterSecureStorage(
    iOptions: IOSOptions(
      accessibility: KeychainAccessibility.first_unlock_this_device,
    ),
  );

  @lazySingleton
  Dio dio(AppConfig config, AuthInterceptor authInterceptor) {
    final dio = Dio(
      BaseOptions(
        baseUrl: config.apiBaseUrl,
        connectTimeout: const Duration(seconds: 10),
        receiveTimeout: const Duration(seconds: 15),
        sendTimeout: const Duration(seconds: 15),
        contentType: Headers.jsonContentType,
        responseType: ResponseType.json,
      ),
    );
    // No LogInterceptor: request/response bodies carry tokens and PII.
    dio.interceptors.add(authInterceptor);
    return dio;
  }
}
