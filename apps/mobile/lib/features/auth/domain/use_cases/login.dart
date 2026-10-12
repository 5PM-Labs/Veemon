import 'package:fpdart/fpdart.dart';
import 'package:injectable/injectable.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/core/utils/validators/email_validator.dart';

import '../entities/user_entity.dart';
import '../repositories/auth_repository.dart';

/// Validates credentials client-side, then signs in.
@injectable
class Login {
  Login(this._repository);

  final AuthRepository _repository;

  Future<Either<Failure, UserEntity>> call({
    required String email,
    required String password,
  }) async {
    final normalizedEmail = email.trim();
    if (!EmailValidator.isValid(normalizedEmail)) {
      return const Left(Failure.validation(ValidationIssue.invalidEmail));
    }
    if (password.isEmpty) {
      return const Left(Failure.validation(ValidationIssue.emptyPassword));
    }
    return _repository.login(email: normalizedEmail, password: password);
  }
}
