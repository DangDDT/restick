import "package:flutter_secure_storage/flutter_secure_storage.dart";

class SecureStorage {
  SecureStorage({FlutterSecureStorage? storage})
      : _storage = storage ??
            const FlutterSecureStorage(
              // wrapKey / wrapKeyIv must be valid base64 when set; leave empty
              // so the web plugin uses its default local key wrapping.
              webOptions: WebOptions(
                dbName: "restick_secure",
                publicKey: "RestickSecureStorage",
              ),
            );

  final FlutterSecureStorage _storage;

  static const String _tokenKey = "access_token";
  static const String _userJsonKey = "user_json";

  Future<String?> readToken() => _storage.read(key: _tokenKey);

  Future<void> writeToken(String token) =>
      _storage.write(key: _tokenKey, value: token);

  Future<void> deleteToken() => _storage.delete(key: _tokenKey);

  Future<String?> readUserJson() => _storage.read(key: _userJsonKey);

  Future<void> writeUserJson(String json) =>
      _storage.write(key: _userJsonKey, value: json);

  Future<void> deleteUserJson() => _storage.delete(key: _userJsonKey);

  Future<void> clearSession() async {
    await deleteToken();
    await deleteUserJson();
  }
}
