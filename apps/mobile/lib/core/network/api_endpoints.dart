/// REST paths declared by the `veemon.route` options in
/// `contract/user/user.proto`. Keep in sync with the contract.
abstract final class ApiEndpoints {
  static const String login = '/api/v1/auth/login';
  static const String register = '/api/v1/auth/register';
  static const String refresh = '/api/v1/auth/refresh';
  static const String me = '/api/v1/auth/me';
  static const String logout = '/api/v1/auth/logout';
}
