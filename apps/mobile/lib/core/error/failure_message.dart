import 'package:flutter/widgets.dart';

import '../language/l10n_extension.dart';
import 'failures.dart';

/// Maps a [Failure] to a localized, actionable message. Technical detail
/// (status codes, server messages) stays in logs, never on screen.
extension FailureMessageX on Failure {
  String message(BuildContext context) {
    final l10n = context.l10n;
    return switch (this) {
      NetworkFailure() => l10n.errorNetwork,
      TimeoutFailure() => l10n.errorTimeout,
      UnauthorizedFailure() => l10n.errorUnauthorized,
      InvalidCredentialsFailure() => l10n.errorInvalidCredentials,
      RateLimitedFailure() => l10n.errorRateLimited,
      ValidationFailure(issue: ValidationIssue.invalidEmail) =>
        l10n.errorInvalidEmail,
      ValidationFailure(issue: ValidationIssue.emptyPassword) =>
        l10n.errorEmptyPassword,
      ServerFailure() => l10n.errorServer,
      UnexpectedFailure() => l10n.errorUnexpected,
    };
  }
}
