import "dart:convert";
import "dart:io";

import "package:dio/dio.dart";
import "package:drift/native.dart";
import "package:flutter_test/flutter_test.dart";
import "package:path/path.dart" as p;
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/storage/prefs_storage.dart";
import "package:restick/src/data/datasources/restick_api.dart";
import "package:restick/src/data/local/app_database.dart";
import "package:restick/src/data/local/pack_downloader.dart";
import "package:restick/src/data/local/pack_file_store.dart";
import "package:restick/src/data/repositories/pack_repository_impl.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:shared_preferences/shared_preferences.dart";

const List<int> _bytes = <int>[1, 2, 3];

Map<String, Object?> _packJson({Object? animated}) {
  return <String, Object?>{
    "schema": "restick.pack.v1",
    "id": "pack-id",
    "version": 2,
    "stickers": <Object?>[
      <String, Object?>{
        "file": "000.webp",
        "thumb": "thumbs/thumb_000.webp",
        "emoji_hint": "🙂",
        "width": 512,
        "height": 512,
        if (animated != null) "animated": animated,
      },
    ],
  };
}

const Map<String, Object?> _animated = <String, Object?>{
  "file": "animated/000.webp",
  "frames": 24,
  "duration_ms": 2000,
  "width": 512,
  "height": 512,
};

Future<PackRepositoryImpl> _repository(RestickDatabase database) async {
  SharedPreferences.setMockInitialValues(<String, Object>{});
  final SharedPreferences prefs = await SharedPreferences.getInstance();
  return PackRepositoryImpl(
    api: RestickApi(Dio()),
    downloadDio: Dio(),
    database: database,
    fileStore: PackFileStore(),
    downloader: PackDownloader(),
    prefs: RestickPrefsStorage(preferences: prefs),
    apiBase: Uri.parse("http://localhost"),
    errorCubit: RestickErrorCubit(),
  );
}

Future<RestickDatabase> _downloadedPack(
  Directory dir, {
  required Map<String, Object?> pack,
}) async {
  File(p.join(dir.path, "pack.json")).writeAsStringSync(jsonEncode(pack));
  File(p.join(dir.path, "000.webp")).writeAsBytesSync(_bytes);
  final RestickDatabase database = await RestickDatabase.open(
    NativeDatabase.memory(),
  );
  await database.upsertPack(
    LocalPacksCompanion.insert(
      id: "pack-id",
      slug: "pack",
      name: "Pack",
      author: "Author",
      tagsJson: "[]",
      stickerCount: 1,
      version: 2,
      thumbnailUrl: "",
      checksum: "",
      status: "downloaded",
      localDir: dir.path,
      updatedAt: DateTime.utc(2026, 11, 13),
    ),
  );
  return database;
}

Future<Directory> _tempDir() async {
  final Directory dir = await Directory.systemTemp.createTemp("restick_pack_");
  addTearDown(() => dir.delete(recursive: true));
  return dir;
}

void main() {
  test("getStickers parses animated entries and old pack.json without animated",
      () async {
    final Directory animatedDir = await _tempDir();
    File(p.join(animatedDir.path, "animated", "000.webp"))
      ..createSync(recursive: true)
      ..writeAsBytesSync(_bytes);
    final RestickDatabase newDatabase = await _downloadedPack(
      animatedDir,
      pack: _packJson(animated: _animated),
    );
    addTearDown(newDatabase.close);

    final List<StickerAsset> fromNew =
        await (await _repository(newDatabase)).getStickers("pack-id");
    expect(fromNew, hasLength(1));
    expect(fromNew.single.isAnimated, isTrue);
    expect(
      fromNew.single.animatedFile!.path,
      p.join(animatedDir.path, "animated", "000.webp"),
    );
    expect(fromNew.single.file.path, p.join(animatedDir.path, "000.webp"));

    final Directory plainDir = await _tempDir();
    final RestickDatabase oldDatabase = await _downloadedPack(
      plainDir,
      pack: _packJson(),
    );
    addTearDown(oldDatabase.close);

    final List<StickerAsset> fromOld =
        await (await _repository(oldDatabase)).getStickers("pack-id");
    expect(fromOld, hasLength(1));
    expect(fromOld.single.isAnimated, isFalse);
    expect(fromOld.single.animatedFile, isNull);
    expect(fromOld.single.file.path, p.join(plainDir.path, "000.webp"));
  });

  test("getStickers falls back to the static file when the animated file is missing",
      () async {
    final Directory dir = await _tempDir();
    final RestickDatabase database = await _downloadedPack(
      dir,
      pack: _packJson(animated: _animated),
    );
    addTearDown(database.close);

    final List<StickerAsset> stickers =
        await (await _repository(database)).getStickers("pack-id");
    expect(stickers.single.isAnimated, isFalse);
    expect(stickers.single.file.path, p.join(dir.path, "000.webp"));
  });

  test("getStickers falls back to the static file on malformed animated metadata",
      () async {
    final Directory dir = await _tempDir();
    File(p.join(dir.path, "animated", "000.webp"))
      ..createSync(recursive: true)
      ..writeAsBytesSync(_bytes);
    final RestickDatabase database = await _downloadedPack(
      dir,
      pack: _packJson(animated: <String, Object?>{"frames": 24}),
    );
    addTearDown(database.close);

    final List<StickerAsset> stickers =
        await (await _repository(database)).getStickers("pack-id");
    expect(stickers.single.isAnimated, isFalse);
    expect(stickers.single.file.path, p.join(dir.path, "000.webp"));
  });

  test("getStickers falls back to the static file on a non-map animated entry",
      () async {
    final Directory dir = await _tempDir();
    final RestickDatabase database = await _downloadedPack(
      dir,
      pack: _packJson(animated: "animated/000.webp"),
    );
    addTearDown(database.close);

    final List<StickerAsset> stickers =
        await (await _repository(database)).getStickers("pack-id");
    expect(stickers.single.isAnimated, isFalse);
    expect(stickers.single.file.path, p.join(dir.path, "000.webp"));
  });
}
