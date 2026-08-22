import "package:drift/native.dart";
import "package:flutter_test/flutter_test.dart";
import "package:restick/restick.dart";

void main() {
  group("RestickConfig", () {
    test("instantiates with baseUrl and apiKey", () {
      const RestickConfig config = RestickConfig(
        baseUrl: "http://localhost",
        apiKey: "test-api-key",
      );
      expect(config.baseUrl, equals("http://localhost"));
      expect(config.apiKey, equals("test-api-key"));
      expect(config.apiBase, equals(Uri.parse("http://localhost")));
      expect(config.databaseExecutor, isNull);
    });

    test("instantiates with custom databaseExecutor", () {
      final QueryExecutor executor = NativeDatabase.memory();
      final RestickConfig config = RestickConfig(
        baseUrl: "http://localhost",
        apiKey: "test-api-key",
        databaseExecutor: executor,
      );
      expect(config.databaseExecutor, equals(executor));
    });

    test("fromJson parses correctly with baseUrl", () {
      final RestickConfig config =
          RestickConfig.fromJson(const <String, Object?>{
        "baseUrl": "https://restick.example.com",
        "apiKey": "rstk_live_123",
        "autoSyncIntervalMinutes": 10,
        "persistApiKey": false,
      });
      expect(config.baseUrl, equals("https://restick.example.com"));
      expect(config.apiKey, equals("rstk_live_123"));
      expect(config.autoSyncInterval, equals(const Duration(minutes: 10)));
      expect(config.persistApiKey, isFalse);
    });

    test("RestickDatabase opens with custom executor", () async {
      final QueryExecutor executor = NativeDatabase.memory();
      final RestickDatabase db = await RestickDatabase.open(executor);
      final dynamic packs = await db.readAllPacks();
      expect(packs, isEmpty);
      await db.close();
    });
  });
}
