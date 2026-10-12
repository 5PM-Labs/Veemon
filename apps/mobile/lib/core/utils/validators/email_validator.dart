/// Pragmatic shape check (`local@domain.tld`); the server stays the authority.
abstract final class EmailValidator {
  static final RegExp _pattern = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

  static bool isValid(String email) => _pattern.hasMatch(email.trim());
}
