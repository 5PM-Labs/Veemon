import 'package:flutter_test/flutter_test.dart';
import 'package:veemon_mobile/core/utils/validators/email_validator.dart';

void main() {
  group('EmailValidator.isValid', () {
    const cases = <(String, String, bool)>[
      ('plain address', 'a@b.co', true),
      ('surrounding whitespace', '  a@b.co ', true),
      ('missing @', 'ab.co', false),
      ('missing tld', 'a@b', false),
      ('inner space', 'a b@c.co', false),
      ('empty', '', false),
    ];
    for (final (name, input, expected) in cases) {
      test(name, () => expect(EmailValidator.isValid(input), expected));
    }
  });
}
