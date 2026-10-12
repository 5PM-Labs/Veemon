import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:veemon_mobile/core/error/failures.dart';

import '../../domain/entities/user_entity.dart';

part 'session_state.freezed.dart';

@freezed
sealed class SessionState with _$SessionState {
  /// Not yet restored (app start).
  const factory SessionState.unknown() = SessionUnknown;

  const factory SessionState.authenticated(UserEntity user) =
      SessionAuthenticated;

  /// [failure] is set when the session ended for a reason worth telling the
  /// user (expired, network trouble during restore); null on a clean logout.
  const factory SessionState.unauthenticated({Failure? failure}) =
      SessionUnauthenticated;
}
