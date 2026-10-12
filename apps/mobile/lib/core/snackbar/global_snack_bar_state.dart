import 'package:freezed_annotation/freezed_annotation.dart';

import '../error/failures.dart';

part 'global_snack_bar_state.freezed.dart';

@freezed
sealed class GlobalSnackBarState with _$GlobalSnackBarState {
  const factory GlobalSnackBarState.idle() = GlobalSnackBarIdle;

  /// [id] makes repeated identical failures distinct states, so each one is
  /// shown.
  const factory GlobalSnackBarState.failure({
    required Failure failure,
    required int id,
  }) = GlobalSnackBarFailure;
}
