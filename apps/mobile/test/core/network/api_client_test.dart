import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:veemon_mobile/core/error/exceptions.dart';
import 'package:veemon_mobile/core/network/api_client.dart';

/// Serves one canned response (or throws one DioException) per request.
class _FakeAdapter implements HttpClientAdapter {
  _FakeAdapter({this.status = 200, this.body, this.error});

  final int status;
  final Object? body;
  final DioExceptionType? error;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (error != null) {
      throw DioException(requestOptions: options, type: error!);
    }
    return ResponseBody.fromString(
      jsonEncode(body),
      status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

ApiClient _client(_FakeAdapter adapter) => ApiClient(
  Dio(BaseOptions(baseUrl: 'http://api.test'))..httpClientAdapter = adapter,
);

void main() {
  group('ApiClient envelope', () {
    test('returns data on a success envelope', () async {
      final client = _client(
        _FakeAdapter(
          body: {
            'success': true,
            'data': {'id': '1'},
          },
        ),
      );
      expect(await client.get('/x'), {'id': '1'});
    });

    test('returns an empty map when data is absent', () async {
      final client = _client(_FakeAdapter(body: {'success': true}));
      expect(await client.post('/x'), isEmpty);
    });

    test('throws ApiException with status + code on an error envelope', () {
      final client = _client(
        _FakeAdapter(
          status: 401,
          body: {
            'success': false,
            'error': {'code': 40101, 'message': 'invalid credentials'},
          },
        ),
      );
      expect(
        client.post('/x'),
        throwsA(
          isA<ApiException>()
              .having((e) => e.statusCode, 'statusCode', 401)
              .having((e) => e.code, 'code', 40101),
        ),
      );
    });

    test('throws MalformedResponseException when data is not an object', () {
      final client = _client(
        _FakeAdapter(body: {'success': true, 'data': 'oops'}),
      );
      expect(client.get('/x'), throwsA(isA<MalformedResponseException>()));
    });
  });

  group('ApiClient transport errors', () {
    final cases = <(DioExceptionType, Matcher)>[
      (DioExceptionType.connectionTimeout, isA<RequestTimeoutException>()),
      (DioExceptionType.receiveTimeout, isA<RequestTimeoutException>()),
      (DioExceptionType.connectionError, isA<NetworkException>()),
      (DioExceptionType.badCertificate, isA<NetworkException>()),
    ];
    for (final (type, matcher) in cases) {
      test('$type', () {
        final client = _client(_FakeAdapter(error: type));
        expect(client.get('/x'), throwsA(matcher));
      });
    }
  });
}
