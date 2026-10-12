/// The single seam for credential storage. Implementations must use the OS
/// keystore — never `shared_preferences` or plain files (MASVS-STORAGE).
abstract interface class TokenStorage {
  Future<String?> read();

  Future<void> write(String token);

  Future<void> clear();
}
