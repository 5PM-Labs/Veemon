import 'package:freezed_annotation/freezed_annotation.dart';

part 'failures.freezed.dart';

/// Why a validation rule rejected user input.
enum ValidationIssue { invalidEmail, emptyPassword }

/// Every operation in domain/presentation returns `Either<Failure, T>`.
/// One subtype per scenario the UI handles differently.
@freezed
sealed class Failure with _$Failure {
  /// Device is offline or the server is unreachable.
  const factory Failure.network() = NetworkFailure;

  /// The request timed out.
  const factory Failure.timeout() = TimeoutFailure;

  /// Session missing, expired, or revoked — the user must sign in again.
  const factory Failure.unauthorized() = UnauthorizedFailure;

  /// Login rejected: wrong email/password.
  const factory Failure.invalidCredentials() = InvalidCredentialsFailure;

  /// Too many attempts (HTTP 429 or account lockout).
  const factory Failure.rateLimited() = RateLimitedFailure;

  /// Client-side input validation failed.
  const factory Failure.validation(ValidationIssue issue) = ValidationFailure;

  /// Any other non-2xx response.
  const factory Failure.server({required int statusCode, int? code}) =
      ServerFailure;

  /// A bug or contract mismatch — log it, show a generic message.
  const factory Failure.unexpected() = UnexpectedFailure;
}
