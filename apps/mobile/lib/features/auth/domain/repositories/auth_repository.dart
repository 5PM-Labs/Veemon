import 'package:fpdart/fpdart.dart';
import 'package:veemon_mobile/core/error/failures.dart';

import '../entities/user_entity.dart';

abstract interface class AuthRepository {
  /// Authenticates and persists the session token on success.
  Future<Either<Failure, UserEntity>> login({
    required String email,
    required String password,
  });

  /// Returns the user for the stored session, or [UnauthorizedFailure] when
  /// there is none (or it was revoked/expired — the token is then cleared).
  Future<Either<Failure, UserEntity>> getCurrentUser();

  /// Revokes the token server-side (best effort) and always clears it locally.
  Future<Either<Failure, Unit>> logout();
}
