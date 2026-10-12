import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:fpdart/fpdart.dart';
import 'package:mockito/mockito.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/login_cubit.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/login_state.dart';

import '../../../../helpers/test_helper.dart';
import '../../helpers/auth_mocks.mocks.dart';
import '../../helpers/auth_test_data.dart';

void main() {
  late MockLogin login;

  setUpAll(registerDummies);

  setUp(() => login = MockLogin());

  test('starts in initial', () {
    expect(LoginCubit(login).state, const LoginState.initial());
  });

  blocTest<LoginCubit, LoginState>(
    'emits [submitting, success] when login succeeds',
    setUp: () =>
        when(login(email: tEmail, password: tPassword))
            .thenAnswer((_) async => Right(tUser)),
    build: () => LoginCubit(login),
    act: (cubit) => cubit.submit(email: tEmail, password: tPassword),
    expect: () => [const LoginState.submitting(), LoginState.success(tUser)],
  );

  blocTest<LoginCubit, LoginState>(
    'emits [submitting, failure] when login fails',
    setUp: () =>
        when(login(email: tEmail, password: tPassword))
            .thenAnswer((_) async => const Left(Failure.invalidCredentials())),
    build: () => LoginCubit(login),
    act: (cubit) => cubit.submit(email: tEmail, password: tPassword),
    expect: () => const [
      LoginState.submitting(),
      LoginState.failure(Failure.invalidCredentials()),
    ],
  );

  blocTest<LoginCubit, LoginState>(
    'ignores a second submit while one is in flight',
    seed: () => const LoginState.submitting(),
    build: () => LoginCubit(login),
    act: (cubit) => cubit.submit(email: tEmail, password: tPassword),
    expect: () => const <LoginState>[],
    verify: (_) => verifyZeroInteractions(login),
  );
}
