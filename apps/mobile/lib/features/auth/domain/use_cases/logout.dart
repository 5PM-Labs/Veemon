import 'package:fpdart/fpdart.dart';
import 'package:injectable/injectable.dart';
import 'package:veemon_mobile/core/error/failures.dart';

import '../repositories/auth_repository.dart';

@injectable
class Logout {
  Logout(this._repository);

  final AuthRepository _repository;

  Future<Either<Failure, Unit>> call() => _repository.logout();
}
