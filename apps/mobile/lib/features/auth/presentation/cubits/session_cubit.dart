import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:fpdart/fpdart.dart';
import 'package:injectable/injectable.dart';
import 'package:veemon_mobile/core/error/failures.dart';

import '../../domain/entities/user_entity.dart';
import '../../domain/use_cases/get_current_user.dart';
import '../../domain/use_cases/logout.dart';
import 'session_state.dart';

/// Global read cubit: who is signed in. Provided once at the app root; the
/// router redirects on its state.
@injectable
class SessionCubit extends Cubit<SessionState> {
  SessionCubit(this._getCurrentUser, this._logout)
    : super(const SessionState.unknown());

  final GetCurrentUser _getCurrentUser;
  final Logout _logout;

  Future<void> restore() async {
    final result = await _getCurrentUser();
    emit(switch (result) {
      Right(:final value) => SessionState.authenticated(value),
      // No stored session is the normal first-launch case — nothing to say.
      Left(value: UnauthorizedFailure()) =>
        const SessionState.unauthenticated(),
      Left(:final value) => SessionState.unauthenticated(failure: value),
    });
  }

  /// Called by the login screen's BlocListener once LoginCubit succeeds.
  void signedIn(UserEntity user) => emit(SessionState.authenticated(user));

  Future<void> logout() async {
    final result = await _logout();
    // The local session is gone either way; report a failed server-side
    // revocation instead of hiding it.
    emit(SessionState.unauthenticated(failure: result.getLeft().toNullable()));
  }
}
