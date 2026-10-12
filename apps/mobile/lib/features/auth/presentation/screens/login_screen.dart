import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:veemon_mobile/core/di/service_locator.dart';
import 'package:veemon_mobile/core/language/l10n_extension.dart';

import '../cubits/login_cubit.dart';
import '../cubits/login_state.dart';
import '../cubits/session_cubit.dart';
import '../widgets/login_form.dart';

class LoginScreen extends StatelessWidget {
  const LoginScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (_) => getIt<LoginCubit>(),
      // Cubits coordinate through the UI layer, never by injecting each other.
      child: BlocListener<LoginCubit, LoginState>(
        listenWhen: (_, current) => current is LoginSuccess,
        listener: (context, state) {
          if (state case LoginSuccess(:final user)) {
            context.read<SessionCubit>().signedIn(user);
          }
        },
        child: Scaffold(
          appBar: AppBar(title: Text(context.l10n.loginTitle)),
          body: const SafeArea(
            child: SingleChildScrollView(
              padding: EdgeInsets.all(24),
              child: LoginForm(),
            ),
          ),
        ),
      ),
    );
  }
}
