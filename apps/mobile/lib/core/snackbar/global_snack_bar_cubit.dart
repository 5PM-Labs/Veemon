import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:injectable/injectable.dart';

import '../error/failures.dart';
import 'global_snack_bar_state.dart';

/// Central channel for surfacing failures to the user — errors are never only
/// logged. The app root listens and renders a SnackBar.
@injectable
class GlobalSnackBarCubit extends Cubit<GlobalSnackBarState> {
  GlobalSnackBarCubit() : super(const GlobalSnackBarState.idle());

  int _nextId = 0;

  void showFailure(Failure failure) =>
      emit(GlobalSnackBarState.failure(failure: failure, id: _nextId++));
}
