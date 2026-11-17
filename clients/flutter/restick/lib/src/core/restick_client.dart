import "dart:async";

import "package:dio/dio.dart";
import "package:restick/src/core/config/restick_config.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/tombstone.dart";
import "package:restick/src/domain/repositories/pack_repository.dart";

class RestickClient {
  RestickClient({
    required PackRepository repository,
    required RestickConfig config,
    required RestickLoadingCubit loadingCubit,
    required RestickErrorCubit errorCubit,
    required List<Dio> dios,
  }) : _repository = repository,
       _config = config,
       _loadingCubit = loadingCubit,
       _errorCubit = errorCubit,
       _dios = dios;

  static const String sdkVersion = "0.1.0";

  final PackRepository _repository;
  final RestickConfig _config;
  final RestickLoadingCubit _loadingCubit;
  final RestickErrorCubit _errorCubit;
  final List<Dio> _dios;
  Timer? _periodicSyncTimer;

  RestickConfig get config => _config;

  RestickLoadingCubit get loadingCubit => _loadingCubit;

  RestickErrorCubit get errorCubit => _errorCubit;

  Future<List<StickerPack>> listMarketplace({String? tag}) {
    return _repository.listMarketplace(tag: tag);
  }

  Future<List<StickerPack>?> cachedMarketplace({String? tag}) {
    return _repository.cachedMarketplace(tag: tag);
  }

  Future<StickerPack> getPack(String packId) {
    return _repository.getPack(packId);
  }

  Future<List<StickerPack>> listGallery() {
    return _repository.listGallery();
  }

  Future<void> downloadPack(String packId) {
    return _repository.downloadPack(packId);
  }

  Future<void> deletePack(String packId) {
    return _repository.deletePack(packId);
  }

  Future<List<StickerAsset>> getStickers(String packId) {
    return _repository.getStickers(packId);
  }

  Stream<PackDownloadState> watchPack(String packId) {
    return _repository.watchPack(packId);
  }

  Stream<PackDownloadState> watchDownloads() {
    return _repository.watchDownloads();
  }

  Future<TombstoneSyncResult> syncRevocations() {
    return _repository.syncRevocations();
  }

  Future<void> selectPack(String packId) {
    return _repository.saveSelectedPackId(packId);
  }

  Future<String?> lastSelectedPackId() {
    return _repository.readSelectedPackId();
  }

  void startPeriodicAutoSync() {
    stopPeriodicAutoSync();
    _periodicSyncTimer = Timer.periodic(_config.autoSyncInterval, (_) {
      unawaited(syncRevocations());
    });
  }

  void stopPeriodicAutoSync() {
    _periodicSyncTimer?.cancel();
    _periodicSyncTimer = null;
  }

  Future<void> dispose() async {
    stopPeriodicAutoSync();
    await _repository.dispose();
    for (final Dio dio in _dios) {
      dio.close();
    }
    await _loadingCubit.close();
    await _errorCubit.close();
  }
}
