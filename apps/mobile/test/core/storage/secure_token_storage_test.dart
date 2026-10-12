import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/mockito.dart';
import 'package:veemon_mobile/core/error/exceptions.dart';
import 'package:veemon_mobile/core/storage/secure_token_storage.dart';

import 'storage_mocks.mocks.dart';

void main() {
  const key = 'veemon.access_token';
  final keystoreError = PlatformException(code: 'BadPaddingException');

  late MockFlutterSecureStorage storage;
  late SecureTokenStorage tokenStorage;

  setUp(() {
    storage = MockFlutterSecureStorage();
    tokenStorage = SecureTokenStorage(storage);
  });

  group('read', () {
    test('returns the stored token', () async {
      when(storage.read(key: key)).thenAnswer((_) async => 'token');

      expect(await tokenStorage.read(), 'token');
    });

    test('treats an unreadable entry as no session and discards it', () async {
      when(storage.read(key: key)).thenThrow(keystoreError);

      expect(await tokenStorage.read(), isNull);
      verify(storage.delete(key: key)).called(1);
    });
  });

  group('write', () {
    test('persists the token', () async {
      await tokenStorage.write('token');

      verify(storage.write(key: key, value: 'token')).called(1);
    });

    test('maps a platform failure to StorageException', () {
      when(storage.write(key: key, value: 'token')).thenThrow(keystoreError);

      expect(tokenStorage.write('token'), throwsA(isA<StorageException>()));
    });
  });

  group('clear', () {
    test('deletes the token', () async {
      await tokenStorage.clear();

      verify(storage.delete(key: key)).called(1);
    });

    test('swallows a platform failure (best effort)', () async {
      when(storage.delete(key: key)).thenThrow(keystoreError);

      await expectLater(tokenStorage.clear(), completes);
    });
  });
}
