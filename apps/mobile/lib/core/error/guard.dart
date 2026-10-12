import 'dart:developer' as developer;

import 'package:fpdart/fpdart.dart';

import 'exceptions.dart';
import 'failures.dart';

/// Runs a data-layer [action] and converts thrown [AppException]s into a
/// [Failure], so repositories never leak exceptions upward.
///
/// [onApiException] lets a repository map endpoint-specific statuses (e.g.
/// 401 on login means invalid credentials, not an expired session).
Future<Either<Failure, T>> guard<T>(
  Future<T> Function() action, {
  Failure? Function(ApiException e)? onApiException,
}) async {
  try {
    return Right(await action());
  } on AppException catch (e, stackTrace) {
    final failure = switch (e) {
      NetworkException() => const Failure.network(),
      RequestTimeoutException() => const Failure.timeout(),
      ApiException() => onApiException?.call(e) ?? mapApiException(e),
      MalformedResponseException() ||
      StorageException() => const Failure.unexpected(),
    };
    // Developer log only: type + status, never request/response bodies.
    developer.log(
      'operation failed: $e',
      name: 'veemon.data',
      error: e,
      stackTrace: stackTrace,
    );
    return Left(failure);
  }
}

/// Default HTTP status → [Failure] mapping for the Go API.
Failure mapApiException(ApiException e) => switch (e.statusCode) {
  401 => const Failure.unauthorized(),
  429 => const Failure.rateLimited(),
  _ => Failure.server(statusCode: e.statusCode, code: e.code),
};
