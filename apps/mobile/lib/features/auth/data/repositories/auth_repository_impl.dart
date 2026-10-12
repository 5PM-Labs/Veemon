import 'package:fpdart/fpdart.dart';
import 'package:injectable/injectable.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/core/error/guard.dart';
import 'package:veemon_mobile/core/storage/token_storage.dart';

import '../../domain/entities/user_entity.dart';
import '../../domain/repositories/auth_repository.dart';
import '../data_sources/auth_remote_data_source.dart';
import '../mappers/user_mapper.dart';

@LazySingleton(as: AuthRepository)
class AuthRepositoryImpl implements AuthRepository {
  AuthRepositoryImpl(this._remote, this._tokenStorage);

  final AuthRemoteDataSource _remote;
  final TokenStorage _tokenStorage;

  @override
  Future<Either<Failure, UserEntity>> login({
    required String email,
    required String password,
  }) => guard(
    () async {
      final response = await _remote.login(email: email, password: password);
      await _tokenStorage.write(response.token);
      return response.user.toEntity();
    },
    // On the login route a 401 means bad credentials, not an expired session.
    onApiException: (e) =>
        e.statusCode == 401 ? const Failure.invalidCredentials() : null,
  );

  @override
  Future<Either<Failure, UserEntity>> getCurrentUser() async {
    final token = await _tokenStorage.read();
    if (token == null || token.isEmpty) {
      return const Left(Failure.unauthorized());
    }
    final result = await guard(() async => (await _remote.me()).toEntity());
    if (result case Left(value: UnauthorizedFailure())) {
      // Expired or revoked server-side — drop the dead credential.
      await _tokenStorage.clear();
    }
    return result;
  }

  @override
  Future<Either<Failure, Unit>> logout() async {
    final result = await guard(_remote.logout);
    // Always clear locally, even if revocation failed (MASVS-STORAGE).
    await _tokenStorage.clear();
    return result.map((_) => unit);
  }
}
