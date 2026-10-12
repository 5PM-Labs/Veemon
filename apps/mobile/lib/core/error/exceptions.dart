/// Exceptions thrown by the data layer (data sources, [ApiClient]).
///
/// They never cross into domain/presentation: repositories catch them and
/// return a `Failure` (see `failures.dart`).
sealed class AppException implements Exception {
  const AppException();
}

/// The device could not reach the server (offline, DNS, TLS, refused).
final class NetworkException extends AppException {
  const NetworkException();
}

/// Connect / send / receive timed out.
final class RequestTimeoutException extends AppException {
  const RequestTimeoutException();
}

/// The server answered with a non-2xx status or a `success: false` envelope.
///
/// [code] is the API's numeric error code from `error.code` (e.g. 40001);
/// [message] is kept for developer logs only — never shown to users verbatim.
final class ApiException extends AppException {
  const ApiException({required this.statusCode, this.code, this.message});

  final int statusCode;
  final int? code;
  final String? message;

  @override
  String toString() => 'ApiException($statusCode, code: $code)';
}

/// Secure on-device storage (Keychain / Keystore) failed to persist data.
final class StorageException extends AppException {
  const StorageException();
}

/// The server answered 2xx but the body didn't match the contract.
final class MalformedResponseException extends AppException {
  const MalformedResponseException(this.reason);

  final String reason;

  @override
  String toString() => 'MalformedResponseException($reason)';
}
