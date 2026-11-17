import "dart:async";

import "package:dio/dio.dart";
import "package:restick/src/core/config/restick_config.dart";
import "package:restick/src/core/error/failure.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/sticker_preview.dart";
import "package:restick/src/domain/entities/tombstone.dart";
import "package:restick/src/domain/repositories/pack_repository.dart";

/// Marketplace repository for widget tests: returns [remote], throws
/// [failure], or never completes when [hang] is set.
class FakeMarketplaceRepository implements PackRepository {
  FakeMarketplaceRepository({
    this.remote = const <StickerPack>[],
    this.failure,
    this.hang = false,
  });

  List<StickerPack> remote;
  Failure? failure;
  bool hang;
  int listCalls = 0;
  final List<String> downloadCalls = <String>[];
  final StreamController<PackDownloadState> downloads =
      StreamController<PackDownloadState>.broadcast();

  @override
  Future<List<StickerPack>?> cachedMarketplace({String? tag}) async => null;

  @override
  Future<List<StickerPack>> listMarketplace({String? tag}) async {
    listCalls++;
    if (hang) {
      return Completer<List<StickerPack>>().future;
    }
    final Failure? error = failure;
    if (error != null) {
      throw error;
    }
    return remote;
  }

  @override
  Stream<PackDownloadState> watchDownloads() => downloads.stream;

  @override
  Future<void> deletePack(String packId) async {}

  @override
  Future<void> dispose() async {
    await downloads.close();
  }

  @override
  Future<void> downloadPack(String packId) async {
    downloadCalls.add(packId);
  }

  @override
  Future<StickerPack> getPack(String packId) async => remote.first;

  @override
  Future<List<StickerAsset>> getStickers(String packId) async =>
      <StickerAsset>[];

  @override
  Future<List<StickerPack>> listGallery() async => <StickerPack>[];

  @override
  Future<String?> readSelectedPackId() async => null;

  @override
  Future<void> saveSelectedPackId(String packId) async {}

  @override
  Future<TombstoneSyncResult> syncRevocations() async =>
      const TombstoneSyncResult(
        purgedPackIds: <String>[],
        tombstones: <Tombstone>[],
      );

  @override
  Stream<PackDownloadState> watchPack(String packId) =>
      const Stream<PackDownloadState>.empty();
}

RestickClient fakeRestickClient(PackRepository repository) {
  return RestickClient(
    repository: repository,
    config: const RestickConfig(baseUrl: "http://localhost", apiKey: "test"),
    loadingCubit: RestickLoadingCubit(),
    errorCubit: RestickErrorCubit(),
    dios: <Dio>[],
  );
}

StickerPack fakePack(
  String id, {
  String? name,
  String author = "Restick",
  List<String> tags = const <String>[],
  int stickerCount = 8,
  String thumbnailUrl = "",
  int previewCount = 3,
  PackDownloadStatus status = PackDownloadStatus.remote,
}) {
  return StickerPack(
    id: id,
    slug: id,
    name: name ?? "Pack $id",
    author: author,
    tags: tags,
    stickerCount: stickerCount,
    version: 1,
    thumbnailUrl: thumbnailUrl,
    checksumSha256: "",
    downloadStatus: status,
    previews: List<StickerPreview>.generate(
      previewCount,
      (_) => const StickerPreview(thumbUrl: "", fileUrl: ""),
    ),
  );
}
