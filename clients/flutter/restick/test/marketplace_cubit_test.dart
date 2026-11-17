import "dart:async";

import "package:dio/dio.dart";
import "package:flutter_test/flutter_test.dart";
import "package:restick/src/core/config/restick_config.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/tombstone.dart";
import "package:restick/src/domain/repositories/pack_repository.dart";
import "package:restick/src/presentation/cubit/marketplace_cubit.dart";

class _FakeRepository implements PackRepository {
  _FakeRepository({this.cached, this.remote = const <StickerPack>[]});

  final List<StickerPack>? cached;
  final List<StickerPack> remote;
  final Completer<List<StickerPack>> remoteResult =
      Completer<List<StickerPack>>();
  final StreamController<PackDownloadState> downloads =
      StreamController<PackDownloadState>.broadcast();

  @override
  Future<List<StickerPack>?> cachedMarketplace({String? tag}) async => cached;

  @override
  Future<List<StickerPack>> listMarketplace({String? tag}) =>
      remoteResult.future;

  @override
  Stream<PackDownloadState> watchDownloads() => downloads.stream;

  @override
  Future<void> deletePack(String packId) async {}

  @override
  Future<void> dispose() async {
    await downloads.close();
  }

  @override
  Future<void> downloadPack(String packId) async {}

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

StickerPack _pack(String id) {
  return StickerPack(
    id: id,
    slug: id,
    name: id,
    author: "author",
    tags: const <String>[],
    stickerCount: 1,
    version: 1,
    thumbnailUrl: "",
    checksumSha256: "",
  );
}

RestickClient _client(_FakeRepository repository) {
  return RestickClient(
    repository: repository,
    config: const RestickConfig(
      baseUrl: "http://localhost",
      apiKey: "test",
    ),
    loadingCubit: RestickLoadingCubit(),
    errorCubit: RestickErrorCubit(),
    dios: <Dio>[],
  );
}

Future<void> _flush() => Future<void>.delayed(Duration.zero);

void main() {
  test("refresh never resurrects absent or revoked download overlays", () async {
    final List<StickerPack> cached = <StickerPack>[_pack("pack")];
    final List<StickerPack> remote = <StickerPack>[_pack("pack")];
    final _FakeRepository repository = _FakeRepository(cached: cached, remote: remote);
    final RestickClient client = _client(repository);
    final MarketplaceCubit cubit = MarketplaceCubit(client: client);
    addTearDown(cubit.close);
    addTearDown(client.dispose);
    repository.remoteResult.complete(remote);
    await cubit.load();
    repository.downloads.add(const PackDownloadState(packId: "pack", status: PackDownloadStatus.revoked));
    await _flush();
    await cubit.load();
    expect(cubit.state.packs, isEmpty);
    cached.clear();
    remote.clear();
    repository.downloads.add(const PackDownloadState(packId: "pack", status: PackDownloadStatus.downloaded, progress: 1));
    await _flush();
    await cubit.load();
    expect(cubit.state.packs, isEmpty);
  });

  test("shows cached packs before the network response", () async {
    final _FakeRepository repository = _FakeRepository(
      cached: <StickerPack>[_pack("cached")],
      remote: <StickerPack>[_pack("remote")],
    );
    final RestickClient client = _client(repository);
    final MarketplaceCubit cubit = MarketplaceCubit(client: client);
    addTearDown(cubit.close);
    addTearDown(client.dispose);

    final Future<void> load = cubit.load();
    await _flush();
    expect(cubit.state.packs.map((StickerPack pack) => pack.id), <String>[
      "cached",
    ]);
    expect(cubit.state.isLoading, isTrue);

    repository.remoteResult.complete(repository.remote);
    await load;
    expect(cubit.state.packs.map((StickerPack pack) => pack.id), <String>[
      "remote",
    ]);
    expect(cubit.state.isLoading, isFalse);
  });

  test("throttles download progress rebuilds to two percent steps", () async {
    final StickerPack pack = _pack("pack");
    final _FakeRepository repository = _FakeRepository(
      remote: <StickerPack>[pack],
    );
    final RestickClient client = _client(repository);
    final MarketplaceCubit cubit = MarketplaceCubit(client: client);
    addTearDown(cubit.close);
    addTearDown(client.dispose);

    repository.remoteResult.complete(<StickerPack>[pack]);
    await cubit.load();

    repository.downloads.add(
      const PackDownloadState(
        packId: "pack",
        status: PackDownloadStatus.downloading,
        progress: 0.01,
      ),
    );
    await _flush();
    expect(cubit.state.packs.single.downloadProgress, 0.01);

    repository.downloads.add(
      const PackDownloadState(
        packId: "pack",
        status: PackDownloadStatus.downloading,
        progress: 0.02,
      ),
    );
    await _flush();
    expect(cubit.state.packs.single.downloadProgress, 0.01);

    repository.downloads.add(
      const PackDownloadState(
        packId: "pack",
        status: PackDownloadStatus.downloading,
        progress: 0.03,
      ),
    );
    await _flush();
    expect(cubit.state.packs.single.downloadProgress, 0.03);
  });
}
