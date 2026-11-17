import "dart:async";

import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/error/failure.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";

class MarketplaceState extends Equatable {
  const MarketplaceState({
    this.packs = const <StickerPack>[],
    this.isLoading = false,
    this.failure,
    this.downloads = const <String, PackDownloadState>{},
  });

  final List<StickerPack> packs;
  final bool isLoading;
  final Failure? failure;
  final Map<String, PackDownloadState> downloads;

  MarketplaceState copyWith({
    List<StickerPack>? packs,
    bool? isLoading,
    Failure? failure,
    Map<String, PackDownloadState>? downloads,
    bool clearFailure = false,
  }) {
    return MarketplaceState(
      packs: packs ?? this.packs,
      isLoading: isLoading ?? this.isLoading,
      failure: clearFailure ? null : failure ?? this.failure,
      downloads: downloads ?? this.downloads,
    );
  }

  @override
  List<Object?> get props => <Object?>[packs, isLoading, failure, downloads];
}

class MarketplaceCubit extends Cubit<MarketplaceState> {
  MarketplaceCubit({required RestickClient client})
    : _client = client,
      super(const MarketplaceState()) {
    _subscription = _client.watchDownloads().listen(_applyDownload);
  }

  final RestickClient _client;
  late final StreamSubscription<PackDownloadState> _subscription;
  String? _tag;
  int _loadToken = 0;
  final Map<String, PackDownloadState> _latestDownloads = <String, PackDownloadState>{};

  Future<void> load({String? tag}) async {
    final String? effectiveTag = tag ?? _tag;
    _tag = effectiveTag;
    final int token = ++_loadToken;
    final List<StickerPack>? cached = await _client.cachedMarketplace(
      tag: effectiveTag,
    );
    if (isClosed || token != _loadToken) {
      return;
    }
    if (cached != null && cached.isNotEmpty) {
      emit(state.copyWith(packs: _overlay(cached), clearFailure: true));
    }
    emit(state.copyWith(isLoading: true, clearFailure: true));
    try {
      final List<StickerPack> packs = await _client.listMarketplace(
        tag: effectiveTag,
      );
      if (isClosed || token != _loadToken) {
        return;
      }
      emit(state.copyWith(packs: _overlay(packs), isLoading: false));
    } on Failure catch (failure) {
      if (isClosed || token != _loadToken) {
        return;
      }
      emit(state.copyWith(isLoading: false, failure: failure));
    }
  }

  Future<void> download(String packId) {
    return _client.downloadPack(packId);
  }

  Future<void> delete(String packId) {
    return _client.deletePack(packId);
  }

  // Metadata controls membership; live operations control download state.
  List<StickerPack> _overlay(List<StickerPack> packs) => <StickerPack>[
    for (final StickerPack pack in packs)
      if (pack.downloadStatus != PackDownloadStatus.revoked &&
          _latestDownloads[pack.id]?.status != PackDownloadStatus.revoked)
        if (_latestDownloads[pack.id] case final PackDownloadState download)
          pack.copyWith(downloadStatus: download.status, downloadProgress: download.progress)
        else
          pack,
  ];

  void _applyDownload(PackDownloadState download) {
    final PackDownloadState? previous = state.downloads[download.packId];
    bool changed = false;
    final List<StickerPack> next =
        state.packs.map((StickerPack pack) {
          if (pack.id != download.packId) {
            return pack;
          }
          // 50 buckets => at most one rebuild per 2% of download progress.
          if (pack.downloadStatus == download.status &&
              (download.status != PackDownloadStatus.downloading ||
                  (pack.downloadProgress * 50).round() ==
                      (download.progress * 50).round())) {
            return pack;
          }
          changed = true;
          return pack.copyWith(
            downloadStatus: download.status,
            downloadProgress: download.progress,
          );
        }).toList();
    if (!changed && previous?.status == download.status && previous?.message == download.message) {
      // Retain exact progress for the next refresh without waking widgets.
      _latestDownloads[download.packId] = download;
      return;
    }
    _latestDownloads[download.packId] = download;
    final Map<String, PackDownloadState> downloads =
        Map<String, PackDownloadState>.of(state.downloads)..[download.packId] = download;
    emit(state.copyWith(packs: next, downloads: downloads, clearFailure: true));
  }

  @override
  Future<void> close() {
    return _subscription.cancel().whenComplete(super.close);
  }
}
