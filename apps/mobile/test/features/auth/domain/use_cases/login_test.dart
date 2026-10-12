import 'package:flutter_test/flutter_test.dart';
import 'package:fpdart/fpdart.dart';
import 'package:mockito/mockito.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/features/auth/domain/use_cases/login.dart';

import '../../../../helpers/test_helper.dart';
import '../../helpers/auth_mocks.mocks.dart';
import '../../helpers/auth_test_data.dart';

void main() {
  late MockAuthRepository repository;
  late Login login;

  setUpAll(registerDummies);

  setUp(() {
    repository = MockAuthRepository();
    login = Login(repository);
  });

  test('delegates trimmed credentials to the repository', () async {
    when(repository.login(email: tEmail, password: tPassword))
        .thenAnswer((_) async => Right(tUser));

    final result = await login(email: '  $tEmail ', password: tPassword);

    expect(result, Right<Failure, Object>(tUser));
    verify(repository.login(email: tEmail, password: tPassword)).called(1);
  });

  test('returns the repository failure unchanged', () async {
    when(repository.login(email: tEmail, password: tPassword))
        .thenAnswer((_) async => const Left(Failure.invalidCredentials()));

    final result = await login(email: tEmail, password: tPassword);

    expect(result, const Left<Failure, Object>(Failure.invalidCredentials()));
  });

  test('rejects an invalid email without calling the repository', () async {
    final result = await login(email: 'not-an-email', password: tPassword);

    expect(
      result,
      const Left<Failure, Object>(
        Failure.validation(ValidationIssue.invalidEmail),
      ),
    );
    verifyZeroInteractions(repository);
  });

  test('rejects an empty password without calling the repository', () async {
    final result = await login(email: tEmail, password: '');

    expect(
      result,
      const Left<Failure, Object>(
        Failure.validation(ValidationIssue.emptyPassword),
      ),
    );
    verifyZeroInteractions(repository);
  });
}
