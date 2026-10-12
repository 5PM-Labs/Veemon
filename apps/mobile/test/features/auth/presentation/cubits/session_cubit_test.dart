import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:fpdart/fpdart.dart';
import 'package:mockito/mockito.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/session_cubit.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/session_state.dart';

import '../../../../helpers/test_helper.dart';
import '../../helpers/auth_mocks.mocks.dart';
import '../../helpers/auth_test_data.dart';

void main() {
  late MockGetCurrentUser getCurrentUser;
  late MockLogout logout;

  SessionCubit build() => SessionCubit(getCurrentUser, logout);

  setUpAll(registerDummies);

  setUp(() {
    getCurrentUser = MockGetCurrentUser();
    logout = MockLogout();
  });

  test('starts unknown', () {
    expect(build().state, const SessionState.unknown());
  });

  group('restore', () {
    blocTest<SessionCubit, SessionState>(
      'authenticates with a valid stored session',
      setUp: () => when(getCurrentUser()).thenAnswer((_) async => Right(tUser)),
      build: build,
      act: (cubit) => cubit.restore(),
      expect: () => [SessionState.authenticated(tUser)],
    );

    blocTest<SessionCubit, SessionState>(
      'is silently unauthenticated when there is no session',
      setUp: () =>
          when(getCurrentUser())
              .thenAnswer((_) async => const Left(Failure.unauthorized())),
      build: build,
      act: (cubit) => cubit.restore(),
      expect: () => const [SessionState.unauthenticated()],
    );

    blocTest<SessionCubit, SessionState>(
      'surfaces other failures to the user',
      setUp: () =>
          when(getCurrentUser())
              .thenAnswer((_) async => const Left(Failure.network())),
      build: build,
      act: (cubit) => cubit.restore(),
      expect: () => const [
        SessionState.unauthenticated(failure: Failure.network()),
      ],
    );
  });

  blocTest<SessionCubit, SessionState>(
    'signedIn authenticates',
    build: build,
    act: (cubit) => cubit.signedIn(tUser),
    expect: () => [SessionState.authenticated(tUser)],
  );

  group('logout', () {
    blocTest<SessionCubit, SessionState>(
      'ends the session cleanly',
      setUp: () => when(logout()).thenAnswer((_) async => const Right(unit)),
      build: build,
      seed: () => SessionState.authenticated(tUser),
      act: (cubit) => cubit.logout(),
      expect: () => const [SessionState.unauthenticated()],
    );

    blocTest<SessionCubit, SessionState>(
      'ends the session and reports a failed revocation',
      setUp: () =>
          when(logout()).thenAnswer((_) async => const Left(Failure.network())),
      build: build,
      seed: () => SessionState.authenticated(tUser),
      act: (cubit) => cubit.logout(),
      expect: () => const [
        SessionState.unauthenticated(failure: Failure.network()),
      ],
    );
  });
}
