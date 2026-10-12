import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:veemon_mobile/core/error/failures.dart';

import '../../domain/entities/user_entity.dart';

part 'login_state.freezed.dart';

@freezed
sealed class LoginState with _$LoginState {
  const factory LoginState.initial() = LoginInitial;

  const factory LoginState.submitting() = LoginSubmitting;

  const factory LoginState.success(UserEntity user) = LoginSuccess;

  const factory LoginState.failure(Failure failure) = LoginFailure;
}
