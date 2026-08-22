import "dart:convert";

import "package:drift/drift.dart";
import "package:drift/wasm.dart";

part "app_database.g.dart";

class CachedPacks extends Table {
  TextColumn get cacheKey => text()();
  TextColumn get payloadJson => text()();
  DateTimeColumn get updatedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => <Column<Object>>{cacheKey};
}

class CachedPackDetails extends Table {
  TextColumn get packId => text()();
  TextColumn get payloadJson => text()();
  DateTimeColumn get updatedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => <Column<Object>>{packId};
}

class CachedRejectionStats extends Table {
  TextColumn get cacheKey => text()();
  TextColumn get payloadJson => text()();
  DateTimeColumn get updatedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => <Column<Object>>{cacheKey};
}

@DriftDatabase(
  tables: <Type>[CachedPacks, CachedPackDetails, CachedRejectionStats],
)
class AppDatabase extends _$AppDatabase {
  AppDatabase(super.e);

  @override
  int get schemaVersion => 1;

  static Future<AppDatabase> open() async {
    final WasmDatabaseResult result = await WasmDatabase.open(
      databaseName: "restick_cache",
      sqlite3Uri: Uri.parse("sqlite3.wasm"),
      driftWorkerUri: Uri.parse("drift_worker.js"),
    );
    return AppDatabase(result.resolvedExecutor);
  }

  Future<void> writePackList(String key, String json) {
    return into(cachedPacks).insertOnConflictUpdate(
      CachedPacksCompanion.insert(
        cacheKey: key,
        payloadJson: json,
        updatedAt: DateTime.now(),
      ),
    );
  }

  Future<String?> readPackList(String key) async {
    final CachedPack? row = await (select(cachedPacks)
          ..where(($CachedPacksTable t) => t.cacheKey.equals(key)))
        .getSingleOrNull();
    return row?.payloadJson;
  }

  Future<void> writePackDetail(String packId, String json) {
    return into(cachedPackDetails).insertOnConflictUpdate(
      CachedPackDetailsCompanion.insert(
        packId: packId,
        payloadJson: json,
        updatedAt: DateTime.now(),
      ),
    );
  }

  Future<String?> readPackDetail(String packId) async {
    final CachedPackDetail? row = await (select(cachedPackDetails)
          ..where(($CachedPackDetailsTable t) => t.packId.equals(packId)))
        .getSingleOrNull();
    return row?.payloadJson;
  }

  Future<void> writeRejectionStats(String json) {
    return into(cachedRejectionStats).insertOnConflictUpdate(
      CachedRejectionStatsCompanion.insert(
        cacheKey: "rejections",
        payloadJson: json,
        updatedAt: DateTime.now(),
      ),
    );
  }

  Future<String?> readRejectionStats() async {
    final CachedRejectionStat? row = await (select(cachedRejectionStats)
          ..where(
            ($CachedRejectionStatsTable t) => t.cacheKey.equals("rejections"),
          ))
        .getSingleOrNull();
    return row?.payloadJson;
  }
}

List<Map<String, Object?>> decodeJsonList(String raw) {
  final Object? decoded = jsonDecode(raw);
  if (decoded is! List) {
    return <Map<String, Object?>>[];
  }
  return decoded
      .map((Object? item) => Map<String, Object?>.from(item! as Map))
      .toList();
}

Map<String, Object?> decodeJsonMap(String raw) {
  return Map<String, Object?>.from(jsonDecode(raw) as Map);
}
