import "dart:io";

import "package:drift/drift.dart";
import "package:drift/native.dart";
import "package:path/path.dart" as p;
import "package:path_provider/path_provider.dart";

part "app_database.g.dart";

class LocalPacks extends Table {
  TextColumn get id => text()();
  TextColumn get slug => text()();
  TextColumn get name => text()();
  TextColumn get author => text()();
  TextColumn get tagsJson => text()();
  IntColumn get stickerCount => integer()();
  IntColumn get version => integer()();
  TextColumn get thumbnailUrl => text()();
  TextColumn get checksum => text()();
  TextColumn get status => text()();
  TextColumn get localDir => text()();
  DateTimeColumn get updatedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => <Column<Object>>{id};
}

class SyncEntries extends Table {
  TextColumn get key => text()();
  TextColumn get value => text()();

  @override
  Set<Column<Object>> get primaryKey => <Column<Object>>{key};
}

@DriftDatabase(tables: <Type>[LocalPacks, SyncEntries])
class RestickDatabase extends _$RestickDatabase {
  RestickDatabase(super.e);

  @override
  int get schemaVersion => 1;

  static const String tombstoneCursorKey = "last_tombstone_at";
  static const String pendingPurgeKey = "pending_purge";

  static Future<RestickDatabase> open([QueryExecutor? executor]) async {
    if (executor != null) {
      return RestickDatabase(executor);
    }
    return RestickDatabase(
      LazyDatabase(() async {
        final Directory docs = await getApplicationDocumentsDirectory();
        final File file = File(p.join(docs.path, "restick", "restick.sqlite"));
        await file.parent.create(recursive: true);
        return NativeDatabase.createInBackground(file);
      }),
    );
  }

  Future<List<LocalPack>> readAllPacks() {
    return select(localPacks).get();
  }

  Future<LocalPack?> readPack(String packId) {
    return (select(localPacks)
          ..where(($LocalPacksTable row) => row.id.equals(packId)))
        .getSingleOrNull();
  }

  Future<List<LocalPack>> readDownloadedPacks() {
    return (select(localPacks)
          ..where(($LocalPacksTable row) => row.status.equals("downloaded")))
        .get();
  }

  Future<void> upsertPack(LocalPacksCompanion pack) {
    return into(localPacks).insertOnConflictUpdate(pack);
  }

  Future<void> deletePack(String packId) {
    return (delete(localPacks)
          ..where(($LocalPacksTable row) => row.id.equals(packId)))
        .go();
  }

  Future<String> readTombstoneCursor() async {
    final SyncEntry? row = await (select(syncEntries)
          ..where(
            ($SyncEntriesTable entry) => entry.key.equals(tombstoneCursorKey),
          ))
        .getSingleOrNull();
    return row?.value ?? "1970-01-01T00:00:00Z";
  }

  Future<void> writeTombstoneCursor(String value) {
    return into(syncEntries).insertOnConflictUpdate(
      SyncEntriesCompanion.insert(key: tombstoneCursorKey, value: value),
    );
  }

  Future<List<String>> readPendingPurge() async {
    final SyncEntry? row = await (select(syncEntries)
          ..where(
            ($SyncEntriesTable entry) => entry.key.equals(pendingPurgeKey),
          ))
        .getSingleOrNull();
    if (row == null || row.value.isEmpty) {
      return <String>[];
    }
    return row.value.split(",").where((String id) => id.isNotEmpty).toList();
  }

  Future<void> writePendingPurge(List<String> ids) {
    return into(syncEntries).insertOnConflictUpdate(
      SyncEntriesCompanion.insert(
        key: pendingPurgeKey,
        value: ids.join(","),
      ),
    );
  }
}
