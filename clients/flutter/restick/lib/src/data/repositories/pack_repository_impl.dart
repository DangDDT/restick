import "dart:async";
import "dart:convert";
import "dart:io";

import "package:dio/dio.dart";
import "package:drift/drift.dart";
import "package:path/path.dart" as p;
import "package:restick/src/core/error/dio_exception_mapper.dart";
import "package:restick/src/core/error/failure.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/storage/prefs_storage.dart";
import "package:restick/src/data/datasources/restick_api.dart";
import "package:restick/src/data/dto/pack_dto.dart";
import "package:restick/src/data/local/app_database.dart";
import "package:restick/src/data/local/pack_downloader.dart";
import "package:restick/src/data/local/pack_file_store.dart";
import "package:restick/src/data/mappers/json_codec.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/tombstone.dart";
import "package:restick/src/domain/repositories/pack_repository.dart";

class PackRepositoryImpl implements PackRepository {
  PackRepositoryImpl({
    required RestickApi api,
    required Dio downloadDio,
    required RestickDatabase database,
    required PackFileStore fileStore,
    required PackDownloader downloader,
    required RestickPrefsStorage prefs,
    required Uri apiBase,
    required RestickErrorCubit errorCubit,
  })  : _api = api,
        _downloadDio = downloadDio,
        _database = database,
        _fileStore = fileStore,
        _downloader = downloader,
        _prefs = prefs,
        _apiBase = apiBase,
        _errorCubit = errorCubit;

  final RestickApi _api;
  final Dio _downloadDio;
  final RestickDatabase _database;
  final PackFileStore _fileStore;
  final PackDownloader _downloader;
  final RestickPrefsStorage _prefs;
  final Uri _apiBase;
  final RestickErrorCubit _errorCubit;
  final StreamController<PackDownloadState> _downloads =
      StreamController<PackDownloadState>.broadcast();
  final Set<String> _downloading = <String>{};

  @override
  Future<List<StickerPack>> listMarketplace({String? tag}) async {
    await syncRevocations();
    try {
      final PackListResponseDto response = await _api.listPacks(tag: tag);
      final Map<String, LocalPack> locals = await _readLocalMap();
      return response.items
          .map((PackSummaryDto dto) => _merge(dto.toEntity(_apiBase), locals))
          .where(
            (StickerPack pack) =>
                pack.downloadStatus != PackDownloadStatus.revoked,
          )
          .toList();
    } on DioException catch (error) {
      throw _throwMapped(error);
    }
  }

  @override
  Future<StickerPack> getPack(String packId) async {
    try {
      final PackSummaryDto dto = await _api.getPack(packId);
      final Map<String, LocalPack> locals = await _readLocalMap();
      return _merge(dto.toEntity(_apiBase), locals);
    } on DioException catch (error) {
      final Failure failure = mapDioException(error, packId: packId);
      if (failure is RevokedFailure) {
        await _purge(packId);
      }
      _errorCubit.report(failure);
      throw failure;
    }
  }

  @override
  Future<List<StickerPack>> listGallery() async {
    await syncRevocations();
    final List<LocalPack> rows = await _database.readDownloadedPacks();
    return rows.map(_fromRow).toList();
  }

  @override
  Future<void> downloadPack(String packId) async {
    if (_downloading.contains(packId)) {
      return;
    }
    _downloading.add(packId);
    _emit(
      PackDownloadState(
        packId: packId,
        status: PackDownloadStatus.downloading,
      ),
    );
    try {
      await syncRevocations();
      final StickerPack remote = await getPack(packId);
      final BundleMetaDto bundle = await _api.getBundle(packId);
      final LocalPack? existing = await _database.readPack(packId);
      if (_isCurrentDownload(existing, bundle)) {
        _emit(
          PackDownloadState(
            packId: packId,
            status: PackDownloadStatus.downloaded,
            progress: 1,
          ),
        );
        return;
      }
      final List<int> zip = await _downloadZip(bundle.rewrittenUrl(_apiBase), packId);
      final Directory dest = _fileStore.packDirectory(
        packId: packId,
        version: bundle.version,
      );
      await _downloader.unzipAndVerify(
        zipBytes: zip,
        dest: dest,
        expectedChecksum: bundle.checksumSha256,
      );
      if (existing != null && existing.localDir != dest.path) {
        await _fileStore.deleteDirectory(existing.localDir);
      }
      await _database.upsertPack(
        _toCompanion(
          remote.copyWith(
            downloadStatus: PackDownloadStatus.downloaded,
            downloadProgress: 1,
            localDir: dest.path,
            version: bundle.version,
            checksumSha256: bundle.checksumSha256,
          ),
        ),
      );
      _emit(
        PackDownloadState(
          packId: packId,
          status: PackDownloadStatus.downloaded,
          progress: 1,
        ),
      );
    } on DioException catch (error) {
      await _handleDownloadFailure(packId, mapDioException(error, packId: packId));
    } on Failure catch (failure) {
      await _handleDownloadFailure(packId, failure);
    } on FormatException catch (error) {
      await _handleDownloadFailure(packId, CacheFailure(error.message));
    } on Object catch (error) {
      await _handleDownloadFailure(packId, UnexpectedFailure(error.toString()));
    } finally {
      _downloading.remove(packId);
    }
  }

  @override
  Future<void> deletePack(String packId) async {
    final LocalPack? cached = await _database.readPack(packId);
    if (cached != null) {
      await _fileStore.deleteDirectory(cached.localDir);
      await _database.deletePack(packId);
    }
    _emit(
      PackDownloadState(
        packId: packId,
        status: PackDownloadStatus.remote,
      ),
    );
  }

  @override
  Future<List<StickerAsset>> getStickers(String packId) async {
    final LocalPack? cached = await _database.readPack(packId);
    if (cached == null || cached.status != PackDownloadStatus.downloaded.name) {
      return <StickerAsset>[];
    }
    final File packFile = File(p.join(cached.localDir, "pack.json"));
    if (!await packFile.exists()) {
      return <StickerAsset>[];
    }
    final Map<String, Object?> pack = asJsonMap(
      jsonDecode(await packFile.readAsString()),
    );
    return asJsonList(pack["stickers"]).map((Object? item) {
      final Map<String, Object?> sticker = asJsonMap(item);
      final String fileName = sticker["file"]! as String;
      final String thumbName = (sticker["thumb"] as String?) ?? "thumbs/thumb_$fileName";
      return StickerAsset(
        packId: packId,
        file: File(p.join(cached.localDir, fileName)),
        thumb: File(
          p.join(
            cached.localDir,
            thumbName.startsWith("thumb_") ? p.join("thumbs", thumbName) : thumbName,
          ),
        ),
        emojiHint: (sticker["emoji_hint"] as String?) ?? "",
        width: (sticker["width"]! as num).toInt(),
        height: (sticker["height"]! as num).toInt(),
      );
    }).toList();
  }

  @override
  Stream<PackDownloadState> watchPack(String packId) async* {
    final LocalPack? cached = await _database.readPack(packId);
    if (cached != null) {
      yield _stateFromRow(cached);
    }
    yield* _downloads.stream.where(
      (PackDownloadState state) => state.packId == packId,
    );
  }

  @override
  Stream<PackDownloadState> watchDownloads() => _downloads.stream;

  @override
  Future<TombstoneSyncResult> syncRevocations() async {
    final List<String> purged = <String>[];
    final List<String> pending = List<String>.from(
      await _database.readPendingPurge(),
    );
    final TombstoneListResponseDto response;
    try {
      response = await _api.listTombstones(
        since: await _database.readTombstoneCursor(),
      );
    } on DioException catch (error) {
      _errorCubit.report(mapDioException(error));
      return const TombstoneSyncResult(
        purgedPackIds: <String>[],
        tombstones: <Tombstone>[],
      );
    }
    String cursor = await _database.readTombstoneCursor();
    final List<Tombstone> items = response.items
        .map((TombstoneDto dto) => dto.toEntity())
        .toList();
    for (final Tombstone item in items) {
      pending.add(item.packId);
      if (item.revokedAt.compareTo(cursor) > 0) {
        cursor = item.revokedAt;
      }
    }
    final List<String> leftover = <String>[];
    for (final String packId in pending.toSet()) {
      try {
        await _purge(packId);
        purged.add(packId);
        _emit(
          PackDownloadState(
            packId: packId,
            status: PackDownloadStatus.revoked,
          ),
        );
      } on Object catch (_) {
        leftover.add(packId);
      }
    }
    await _database.writePendingPurge(leftover);
    await _database.writeTombstoneCursor(cursor);
    return TombstoneSyncResult(purgedPackIds: purged, tombstones: items);
  }

  @override
  Future<void> saveSelectedPackId(String packId) {
    return _prefs.writeSelectedPackId(packId);
  }

  @override
  Future<String?> readSelectedPackId() async {
    return _prefs.readSelectedPackId();
  }

  @override
  Future<void> dispose() async {
    await _downloads.close();
    await _database.close();
  }

  Future<Map<String, LocalPack>> _readLocalMap() async {
    final List<LocalPack> rows = await _database.readAllPacks();
    return <String, LocalPack>{
      for (final LocalPack row in rows) row.id: row,
    };
  }

  StickerPack _merge(StickerPack remote, Map<String, LocalPack> locals) {
    final LocalPack? local = locals[remote.id];
    if (local == null) {
      return remote;
    }
    return remote.copyWith(
      downloadStatus: _statusFromName(local.status),
      localDir: local.localDir,
    );
  }

  StickerPack _fromRow(LocalPack row) {
    return StickerPack(
      id: row.id,
      slug: row.slug,
      name: row.name,
      author: row.author,
      tags: asStringList(jsonDecode(row.tagsJson)),
      stickerCount: row.stickerCount,
      version: row.version,
      thumbnailUrl: row.thumbnailUrl,
      checksumSha256: row.checksum,
      downloadStatus: _statusFromName(row.status),
      downloadProgress:
          row.status == PackDownloadStatus.downloaded.name ? 1 : 0,
      localDir: row.localDir,
    );
  }

  LocalPacksCompanion _toCompanion(StickerPack pack) {
    return LocalPacksCompanion(
      id: Value<String>(pack.id),
      slug: Value<String>(pack.slug),
      name: Value<String>(pack.name),
      author: Value<String>(pack.author),
      tagsJson: Value<String>(jsonEncode(pack.tags)),
      stickerCount: Value<int>(pack.stickerCount),
      version: Value<int>(pack.version),
      thumbnailUrl: Value<String>(pack.thumbnailUrl),
      checksum: Value<String>(pack.checksumSha256),
      status: Value<String>(pack.downloadStatus.name),
      localDir: Value<String>(pack.localDir),
      updatedAt: Value<DateTime>(DateTime.now()),
    );
  }

  PackDownloadStatus _statusFromName(String name) {
    return PackDownloadStatus.values.firstWhere(
      (PackDownloadStatus status) => status.name == name,
      orElse: () => PackDownloadStatus.remote,
    );
  }

  PackDownloadState _stateFromRow(LocalPack row) {
    final PackDownloadStatus status = _statusFromName(row.status);
    return PackDownloadState(
      packId: row.id,
      status: status,
      progress: status == PackDownloadStatus.downloaded ? 1 : 0,
    );
  }

  bool _isCurrentDownload(LocalPack? existing, BundleMetaDto bundle) {
    return existing != null &&
        existing.version == bundle.version &&
        existing.checksum == bundle.checksumSha256 &&
        existing.status == PackDownloadStatus.downloaded.name;
  }

  Future<List<int>> _downloadZip(String url, String packId) async {
    final Response<List<int>> response = await _downloadDio.get<List<int>>(
      url,
      options: Options(responseType: ResponseType.bytes),
      onReceiveProgress: (int count, int total) {
        if (total <= 0) {
          return;
        }
        _emit(
          PackDownloadState(
            packId: packId,
            status: PackDownloadStatus.downloading,
            progress: count / total,
          ),
        );
      },
    );
    return response.data ?? <int>[];
  }

  Future<void> _purge(String packId) async {
    final LocalPack? cached = await _database.readPack(packId);
    if (cached != null) {
      await _fileStore.deleteDirectory(cached.localDir);
      await _database.deletePack(packId);
    }
  }

  Future<void> _handleDownloadFailure(String packId, Failure failure) async {
    if (failure is RevokedFailure) {
      await _purge(packId);
      _emit(
        PackDownloadState(
          packId: packId,
          status: PackDownloadStatus.revoked,
        ),
      );
    } else {
      _emit(
        PackDownloadState(
          packId: packId,
          status: PackDownloadStatus.failed,
          message: failure.message,
        ),
      );
    }
    _errorCubit.report(failure);
  }

  Failure _throwMapped(DioException error, {String packId = ""}) {
    final Failure failure = mapDioException(error, packId: packId);
    _errorCubit.report(failure);
    return failure;
  }

  void _emit(PackDownloadState state) {
    if (!_downloads.isClosed) {
      _downloads.add(state);
    }
  }
}
