import 'package:dio/dio.dart';
import 'package:injectable/injectable.dart';

import '../error/exceptions.dart';

typedef JsonMap = Map<String, dynamic>;

/// Thin wrapper over [Dio] that unwraps the Go API's `pkg/response` envelope:
///
///   success: `{ "success": true, "data": {...}, "meta": {...}? }`
///   error:   `{ "success": false, "error": { "code": 40001, "message": "..." } }`
///
/// Returns the `data` object and throws an [AppException] otherwise.
@lazySingleton
class ApiClient {
  ApiClient(this._dio);

  final Dio _dio;

  Future<JsonMap> get(String path) => _send(() => _dio.get<Object?>(path));

  Future<JsonMap> post(String path, {JsonMap? body}) =>
      _send(() => _dio.post<Object?>(path, data: body));

  Future<JsonMap> _send(Future<Response<Object?>> Function() request) async {
    final Response<Object?> response;
    try {
      response = await request();
    } on DioException catch (e) {
      throw _mapDioException(e);
    }
    return _unwrap(response.data, response.statusCode ?? 0);
  }

  JsonMap _unwrap(Object? body, int statusCode) {
    if (body is! JsonMap || body['success'] != true) {
      throw _apiExceptionFrom(body, statusCode);
    }
    final data = body['data'];
    return switch (data) {
      final JsonMap map => map,
      null => const <String, dynamic>{},
      _ => throw const MalformedResponseException('data is not an object'),
    };
  }

  AppException _mapDioException(DioException e) => switch (e.type) {
    DioExceptionType.connectionTimeout ||
    DioExceptionType.sendTimeout ||
    DioExceptionType.receiveTimeout ||
    DioExceptionType.transformTimeout => const RequestTimeoutException(),
    DioExceptionType.connectionError ||
    DioExceptionType.badCertificate => const NetworkException(),
    DioExceptionType.badResponse => _apiExceptionFrom(
      e.response?.data,
      e.response?.statusCode ?? 0,
    ),
    DioExceptionType.cancel ||
    DioExceptionType.unknown => const NetworkException(),
  };

  ApiException _apiExceptionFrom(Object? body, int statusCode) {
    final error = body is JsonMap ? body['error'] : null;
    return ApiException(
      statusCode: statusCode,
      code: error is JsonMap ? (error['code'] as num?)?.toInt() : null,
      message: error is JsonMap ? error['message'] as String? : null,
    );
  }
}
