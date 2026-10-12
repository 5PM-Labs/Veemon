import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:fpdart/fpdart.dart';
import 'package:injectable/injectable.dart';

import '../../domain/use_cases/login.dart';
import 'login_state.dart';

/// Write cubit scoped to the login screen.
@injectable
class LoginCubit extends Cubit<LoginState> {
  LoginCubit(this._login) : super(const LoginState.initial());

  final Login _login;

  Future<void> submit({required String email, required String password}) async {
    if (state is LoginSubmitting) return;
    emit(const LoginState.submitting());
    final result = await _login(email: email, password: password);
    emit(switch (result) {
      Right(:final value) => LoginState.success(value),
      Left(:final value) => LoginState.failure(value),
    });
  }
}
