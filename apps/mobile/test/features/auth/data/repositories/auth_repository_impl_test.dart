import 'package:flutter_test/flutter_test.dart';
import 'package:fpdart/fpdart.dart';
import 'package:mockito/mockito.dart';
import 'package:veemon_mobile/core/error/exceptions.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/features/auth/data/repositories/auth_repository_impl.dart';

import '../../helpers/auth_mocks.mocks.dart';
import '../../helpers/auth_test_data.dart';

void main() {
  late MockAuthRemoteDataSource remote;
  late MockTokenStorage tokenStorage;
  late AuthRepositoryImpl repository;

  setUp(() {
    remote = MockAuthRemoteDataSource();
    tokenStorage = MockTokenStorage();
    repository = AuthRepositoryImpl(remote, tokenStorage);
  });

  group('login', () {
    test('stores the token and returns the mapped user', () async {
      when(remote.login(email: tEmail, password: tPassword))
          .thenAnswer((_) async => tLoginResponse);

      final result = await repository.login(email: tEmail, password: tPassword);

      expect(result, Right<Failure, Object>(tUser));
      verify(tokenStorage.write(tToken)).called(1);
    });

    final cases = <(String, AppException, Failure)>[
      (
        '401 → invalid credentials',
        const ApiException(statusCode: 401),
        const Failure.invalidCredentials(),
      ),
      (
        '429 → rate limited',
        const ApiException(statusCode: 429),
        const Failure.rateLimited(),
      ),
      (
        '500 → server',
        const ApiException(statusCode: 500, code: 50001),
        const Failure.server(statusCode: 500, code: 50001),
      ),
      ('offline → network', const NetworkException(), const Failure.network()),
      (
        'timeout → timeout',
        const RequestTimeoutException(),
        const Failure.timeout(),
      ),
    ];
    for (final (name, exception, failure) in cases) {
      test(name, () async {
        when(remote.login(email: tEmail, password: tPassword))
            .thenThrow(exception);

        final result = await repository.login(
          email: tEmail,
          password: tPassword,
        );

        expect(result, Left<Failure, Object>(failure));
        verifyNever(tokenStorage.write(any));
      });
    }
  });

  group('getCurrentUser', () {
    test('is unauthorized without calling the API when no token', () async {
      when(tokenStorage.read()).thenAnswer((_) async => null);

      final result = await repository.getCurrentUser();

      expect(result, const Left<Failure, Object>(Failure.unauthorized()));
      verifyZeroInteractions(remote);
    });

    test('returns the user for a stored token', () async {
      when(tokenStorage.read()).thenAnswer((_) async => tToken);
      when(remote.me()).thenAnswer((_) async => tUserDto);

      expect(await repository.getCurrentUser(), Right<Failure, Object>(tUser));
    });

    test('clears a token the server rejects', () async {
      when(tokenStorage.read()).thenAnswer((_) async => tToken);
      when(remote.me()).thenThrow(const ApiException(statusCode: 401));

      final result = await repository.getCurrentUser();

      expect(result, const Left<Failure, Object>(Failure.unauthorized()));
      verify(tokenStorage.clear()).called(1);
    });

    test('keeps the token on a network failure', () async {
      when(tokenStorage.read()).thenAnswer((_) async => tToken);
      when(remote.me()).thenThrow(const NetworkException());

      final result = await repository.getCurrentUser();

      expect(result, const Left<Failure, Object>(Failure.network()));
      verifyNever(tokenStorage.clear());
    });
  });

  group('logout', () {
    test('clears the token after server revocation', () async {
      when(remote.logout()).thenAnswer((_) async {});

      expect(await repository.logout(), const Right<Failure, Unit>(unit));
      verify(tokenStorage.clear()).called(1);
    });

    test('still clears the token when revocation fails', () async {
      when(remote.logout()).thenThrow(const NetworkException());

      expect(
        await repository.logout(),
        const Left<Failure, Unit>(Failure.network()),
      );
      verify(tokenStorage.clear()).called(1);
    });
  });
}
