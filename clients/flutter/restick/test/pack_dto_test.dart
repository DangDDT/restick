import "dart:convert";
import "dart:typed_data";

import "package:dio/dio.dart";
import "package:drift/native.dart";
import "package:flutter_test/flutter_test.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/storage/prefs_storage.dart";
import "package:restick/src/data/datasources/restick_api.dart";
import "package:restick/src/data/dto/pack_dto.dart";
import "package:restick/src/data/local/app_database.dart";
import "package:restick/src/data/local/pack_downloader.dart";
import "package:restick/src/data/local/pack_file_store.dart";
import "package:restick/src/data/repositories/pack_repository_impl.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/sticker_preview.dart";
import "package:shared_preferences/shared_preferences.dart";

final Uri _apiBase = Uri.parse("https://api.example.com");

Map<String, dynamic> _json({Map<String, dynamic>? extra}) {
  return <String, dynamic>{
    "id": "pack-id",
    "slug": "pack",
    "name": "Pack",
    "author": "Author",
    "tags": <String>[],
    "sticker_count": 3,
    "version": 1,
    "thumbnail_url": "",
    "checksum_sha256": "",
    ...?extra,
  };
}

class _FakeAdapter implements HttpClientAdapter {
  _FakeAdapter(this.body);

  final Map<String, dynamic> body;

  @override
  void close({bool force = false}) {}

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    return ResponseBody.fromString(
      jsonEncode(body),
      200,
      headers: <String, List<String>>{
        Headers.contentTypeHeader: <String>[Headers.jsonContentType],
      },
    );
  }
}

void main() {
  test("parses preview_stickers and theme_color", () {
    final StickerPack pack = PackSummaryDto.fromJson(
      _json(
        extra: <String, dynamic>{
          "theme_color": "#FFAA00",
          "preview_stickers": <Object?>[
            <String, dynamic>{"thumb_url": "https://c/t1", "file_url": "https://c/f1"},
            <String, dynamic>{"thumb_url": "https://c/t2", "file_url": "https://c/f2"},
          ],
        },
      ),
    ).toEntity(_apiBase);
    expect(pack.themeColor, "#FFAA00");
    expect(pack.previews, const <StickerPreview>[
      StickerPreview(thumbUrl: "https://c/t1", fileUrl: "https://c/f1"),
      StickerPreview(thumbUrl: "https://c/t2", fileUrl: "https://c/f2"),
    ]);
  });

  test("server without new fields yields empty previews and empty color", () {
    for (final Map<String, dynamic> extra in <Map<String, dynamic>>[
      <String, dynamic>{},
      <String, dynamic>{"theme_color": null, "preview_stickers": null},
    ]) {
      final StickerPack pack =
          PackSummaryDto.fromJson(_json(extra: extra)).toEntity(_apiBase);
      expect(pack.previews, isEmpty);
      expect(pack.themeColor, "");
    }
  });

  test("malformed preview entries are skipped", () {
    final StickerPack pack = PackSummaryDto.fromJson(
      _json(
        extra: <String, dynamic>{
          "preview_stickers": <Object?>[
            "nope",
            null,
            <String, dynamic>{"thumb_url": "https://c/t"},
            <String, dynamic>{"thumb_url": 1, "file_url": "https://c/f"},
            <String, dynamic>{"thumb_url": "https://c/t", "file_url": "https://c/f"},
          ],
        },
      ),
    ).toEntity(_apiBase);
    expect(pack.previews, const <StickerPreview>[
      StickerPreview(thumbUrl: "https://c/t", fileUrl: "https://c/f"),
    ]);
  });

  test("preview urls are rewritten against apiBase", () {
    final StickerPack pack = PackSummaryDto.fromJson(
      _json(
        extra: <String, dynamic>{
          "preview_stickers": <Object?>[
            <String, dynamic>{
              "thumb_url": "http://localhost:8080/a/t.webp",
              "file_url": "http://127.0.0.1:8080/a/f.webp",
            },
          ],
        },
      ),
    ).toEntity(_apiBase);
    expect(pack.previews.single.thumbUrl, "https://api.example.com/a/t.webp");
    expect(pack.previews.single.fileUrl, "https://api.example.com/a/f.webp");
  });

  test("copyWith keeps previews and themeColor", () {
    const StickerPack pack = StickerPack(
      id: "i",
      slug: "s",
      name: "n",
      author: "a",
      tags: <String>[],
      stickerCount: 1,
      version: 1,
      thumbnailUrl: "",
      checksumSha256: "",
      themeColor: "#112233",
      previews: <StickerPreview>[StickerPreview(thumbUrl: "t", fileUrl: "f")],
    );
    final StickerPack copy = pack.copyWith(name: "other");
    expect(copy.themeColor, "#112233");
    expect(copy.previews, pack.previews);
    expect(copy, isNot(pack));
  });

  test("merge keeps remote previews for a locally downloaded pack", () async {
    SharedPreferences.setMockInitialValues(<String, Object>{});
    final SharedPreferences prefs = await SharedPreferences.getInstance();
    final RestickDatabase database =
        await RestickDatabase.open(NativeDatabase.memory());
    await database.upsertPack(
      LocalPacksCompanion.insert(
        id: "pack-id",
        slug: "pack",
        name: "Pack",
        author: "Author",
        tagsJson: "[]",
        stickerCount: 1,
        version: 1,
        thumbnailUrl: "",
        checksum: "",
        status: "downloaded",
        localDir: "/tmp/x",
        updatedAt: DateTime.utc(2026, 11, 13),
      ),
    );
    final Dio dio = Dio(BaseOptions(baseUrl: "https://api.example.com"))
      ..httpClientAdapter = _FakeAdapter(<String, dynamic>{
        "items": <Object?>[
          _json(
            extra: <String, dynamic>{
              "theme_color": "#FFAA00",
              "preview_stickers": <Object?>[
                <String, dynamic>{"thumb_url": "https://c/t", "file_url": "https://c/f"},
              ],
            },
          ),
        ],
      });
    final PackRepositoryImpl repository = PackRepositoryImpl(
      api: RestickApi(dio),
      downloadDio: Dio(),
      database: database,
      fileStore: PackFileStore(),
      downloader: PackDownloader(),
      prefs: RestickPrefsStorage(preferences: prefs),
      apiBase: _apiBase,
      errorCubit: RestickErrorCubit(),
    );
    await repository.listMarketplace();
    final List<StickerPack> merged = (await repository.cachedMarketplace())!;
    expect(merged.single.downloadStatus, PackDownloadStatus.downloaded);
    expect(merged.single.themeColor, "#FFAA00");
    expect(merged.single.previews.single.fileUrl, "https://c/f");
  });
}
