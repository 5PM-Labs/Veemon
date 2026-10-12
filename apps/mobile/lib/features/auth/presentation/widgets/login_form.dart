import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:veemon_mobile/core/error/failure_message.dart';
import 'package:veemon_mobile/core/language/l10n_extension.dart';

import '../cubits/login_cubit.dart';
import '../cubits/login_state.dart';

class LoginForm extends StatefulWidget {
  const LoginForm({super.key});

  @override
  State<LoginForm> createState() => _LoginFormState();
}

class _LoginFormState extends State<LoginForm> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _submit() {
    FocusScope.of(context).unfocus();
    context.read<LoginCubit>().submit(
      email: _emailController.text,
      password: _passwordController.text,
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return AutofillGroup(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _emailController,
            decoration: InputDecoration(labelText: l10n.loginEmailLabel),
            keyboardType: TextInputType.emailAddress,
            autofillHints: const [AutofillHints.email],
            autocorrect: false,
            textInputAction: TextInputAction.next,
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _passwordController,
            decoration: InputDecoration(labelText: l10n.loginPasswordLabel),
            obscureText: true,
            enableSuggestions: false,
            autocorrect: false,
            autofillHints: const [AutofillHints.password],
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _submit(),
          ),
          const SizedBox(height: 16),
          const _LoginError(),
          const SizedBox(height: 8),
          _SubmitButton(onPressed: _submit),
        ],
      ),
    );
  }
}

class _LoginError extends StatelessWidget {
  const _LoginError();

  @override
  Widget build(BuildContext context) {
    final failure = context.select<LoginCubit, LoginFailure?>(
      (cubit) => switch (cubit.state) {
        final LoginFailure state => state,
        _ => null,
      },
    );
    if (failure == null) return const SizedBox.shrink();
    return Text(
      failure.failure.message(context),
      style: TextStyle(color: Theme.of(context).colorScheme.error),
    );
  }
}

class _SubmitButton extends StatelessWidget {
  const _SubmitButton({required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final isSubmitting = context.select<LoginCubit, bool>(
      (cubit) => cubit.state is LoginSubmitting || cubit.state is LoginSuccess,
    );
    return FilledButton(
      onPressed: isSubmitting ? null : onPressed,
      child: isSubmitting
          ? const SizedBox.square(
              dimension: 20,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : Text(context.l10n.loginSubmit),
    );
  }
}
