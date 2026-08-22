import "package:flutter_secure_storage/flutter_secure_storage.dart";

class RestickSecureStorage {
  RestickSecureStorage({FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage();

  static const String apiKeyKey = "restick.api_key";

  final FlutterSecureStorage _storage;

  Future<void> writeApiKey(String apiKey) {
    return _storage.write(key: apiKeyKey, value: apiKey);
  }

  Future<String?> readApiKey() {
    return _storage.read(key: apiKeyKey);
  }

  Future<void> deleteApiKey() {
    return _storage.delete(key: apiKeyKey);
  }
}
