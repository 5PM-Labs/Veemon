import 'dart:developer' as developer;

import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:injectable/injectable.dart';

import '../error/exceptions.dart';
import 'token_storage.dart';

/// Keystore/Keychain-backed [TokenStorage].
///
/// Platform storage can fail (e.g. an Android Keystore key invalidated after
/// an OS upgrade or device restore throws `BadPaddingException`). Those
/// failures must never escape as a raw [PlatformException]: callers would
/// hang mid-flow (splash screen, submitting state).
@LazySingleton(as: TokenStorage)
class SecureTokenStorage implements TokenStorage {
  SecureTokenStorage(this._storage);

  static const String _key = 'veemon.access_token';

  final FlutterSecureStorage _storage;

  /// An unreadable entry is treated as "no session": it is discarded so the
  /// user can sign in again instead of being locked out.
  @override
  Future<String?> read() async {
    try {
      return await _storage.read(key: _key);
    } on PlatformException catch (e, stackTrace) {
      _log('read failed; discarding unreadable token', e, stackTrace);
      await clear();
      return null;
    }
  }

  @override
  Future<void> write(String token) async {
    try {
      await _storage.write(key: _key, value: token);
    } on PlatformException catch (e, stackTrace) {
      _log('write failed', e, stackTrace);
      throw const StorageException();
    }
  }

  /// Best effort: there is nothing further to fall back to if deletion fails.
  @override
  Future<void> clear() async {
    try {
      await _storage.delete(key: _key);
    } on PlatformException catch (e, stackTrace) {
      _log('clear failed', e, stackTrace);
    }
  }

  // Logs the platform error code only — never the token value.
  void _log(String message, PlatformException e, StackTrace stackTrace) =>
      developer.log(
        '$message (${e.code})',
        name: 'veemon.storage',
        stackTrace: stackTrace,
      );
}
