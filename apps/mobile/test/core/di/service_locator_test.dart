import 'package:flutter_test/flutter_test.dart';
import 'package:veemon_mobile/core/di/service_locator.dart';
import 'package:veemon_mobile/core/snackbar/global_snack_bar_cubit.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/login_cubit.dart';
import 'package:veemon_mobile/features/auth/presentation/cubits/session_cubit.dart';

/// Catches missing `@injectable` annotations / un-regenerated config: every
/// cubit the UI asks get_it for must resolve its whole dependency graph.
void main() {
  setUpAll(configureDependencies);
  tearDownAll(getIt.reset);

  test('resolves every cubit the UI requests', () async {
    final cubits = [
      getIt<SessionCubit>(),
      getIt<LoginCubit>(),
      getIt<GlobalSnackBarCubit>(),
    ];
    for (final cubit in cubits) {
      await cubit.close();
    }
  });
}
