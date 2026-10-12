import 'package:fpdart/fpdart.dart';
import 'package:injectable/injectable.dart';
import 'package:veemon_mobile/core/error/failures.dart';

import '../entities/user_entity.dart';
import '../repositories/auth_repository.dart';

/// Restores the session on app start.
@injectable
class GetCurrentUser {
  GetCurrentUser(this._repository);

  final AuthRepository _repository;

  Future<Either<Failure, UserEntity>> call() => _repository.getCurrentUser();
}
