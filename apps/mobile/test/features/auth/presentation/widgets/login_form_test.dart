import 'package:bloc_test/bloc_test.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/core/language/gen/app_localizations.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/login_cubit.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/login_state.dart';
import 'package:veemon_mobile/features/auth/presentation/widgets/login_form.dart';

class MockLoginCubit extends MockCubit<LoginState> implements LoginCubit {}

/// Records submits without touching use cases. bloc_test's MockCubit is
/// mocktail-based; a fake avoids a second mocking library next to mockito.
class _RecordingLoginCubit extends Cubit<LoginState> implements LoginCubit {
  _RecordingLoginCubit() : super(const LoginState.initial());

  final submitted = <(String, String)>[];

  @override
  Future<void> submit({
    required String email,
    required String password,
  }) async => submitted.add((email, password));
}

Future<void> _pump(WidgetTester tester, LoginCubit cubit) => tester.pumpWidget(
  MaterialApp(
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: BlocProvider<LoginCubit>.value(
      value: cubit,
      child: const Scaffold(body: LoginForm()),
    ),
  ),
);

void main() {
  late MockLoginCubit cubit;

  setUp(() => cubit = MockLoginCubit());

  testWidgets('shows the localized error for invalid credentials', (
    tester,
  ) async {
    whenListen(
      cubit,
      const Stream<LoginState>.empty(),
      initialState: const LoginState.failure(Failure.invalidCredentials()),
    );

    await _pump(tester, cubit);

    expect(find.text('Incorrect email or password.'), findsOneWidget);
  });

  testWidgets('disables submit while submitting', (tester) async {
    whenListen(
      cubit,
      const Stream<LoginState>.empty(),
      initialState: const LoginState.submitting(),
    );

    await _pump(tester, cubit);

    final button = tester.widget<FilledButton>(find.byType(FilledButton));
    expect(button.onPressed, isNull);
  });

  testWidgets('submits the entered credentials', (tester) async {
    final recording = _RecordingLoginCubit();
    addTearDown(recording.close);

    await _pump(tester, recording);
    await tester.enterText(find.byType(TextField).first, 'jane@example.test');
    await tester.enterText(find.byType(TextField).last, 'pw');
    await tester.tap(find.byType(FilledButton));

    expect(recording.submitted, [('jane@example.test', 'pw')]);
  });
}
