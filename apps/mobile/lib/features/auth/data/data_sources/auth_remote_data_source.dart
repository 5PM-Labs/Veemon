import 'package:injectable/injectable.dart';
import 'package:veemon_mobile/core/error/exceptions.dart';
import 'package:veemon_mobile/core/network/api_client.dart';
import 'package:veemon_mobile/core/network/api_endpoints.dart';

import '../dtos/login_response_dto.dart';
import '../dtos/user_dto.dart';

/// Calls the Go API's auth routes. Throws [AppException]s; the repository
/// converts them to failures.
abstract interface class AuthRemoteDataSource {
  Future<LoginResponseDto> login({
    required String email,
    required String password,
  });

  Future<UserDto> me();

  Future<void> logout();
}

@LazySingleton(as: AuthRemoteDataSource)
class AuthRemoteDataSourceImpl implements AuthRemoteDataSource {
  AuthRemoteDataSourceImpl(this._client);

  final ApiClient _client;

  @override
  Future<LoginResponseDto> login({
    required String email,
    required String password,
  }) async {
    final data = await _client.post(
      ApiEndpoints.login,
      body: {'email': email, 'password': password},
    );
    return _parse(() => LoginResponseDto.fromJson(data));
  }

  @override
  Future<UserDto> me() async {
    final data = await _client.get(ApiEndpoints.me);
    return _parse(() => UserDto.fromJson(data));
  }

  @override
  Future<void> logout() => _client.post(ApiEndpoints.logout);

  T _parse<T>(T Function() fromJson) {
    try {
      return fromJson();
    } on Object catch (e) {
      // json_serializable throws TypeError/CheckedFromJsonException on a
      // contract mismatch; surface it as a data-layer exception.
      throw MalformedResponseException(e.runtimeType.toString());
    }
  }
}
