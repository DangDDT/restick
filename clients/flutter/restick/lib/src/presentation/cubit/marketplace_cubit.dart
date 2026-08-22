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
  });

  final List<StickerPack> packs;
  final bool isLoading;
  final Failure? failure;

  MarketplaceState copyWith({
    List<StickerPack>? packs,
    bool? isLoading,
    Failure? failure,
    bool clearFailure = false,
  }) {
    return MarketplaceState(
      packs: packs ?? this.packs,
      isLoading: isLoading ?? this.isLoading,
      failure: clearFailure ? null : failure ?? this.failure,
    );
  }

  @override
  List<Object?> get props => <Object?>[packs, isLoading, failure];
}

class MarketplaceCubit extends Cubit<MarketplaceState> {
  MarketplaceCubit({required RestickClient client})
      : _client = client,
        super(const MarketplaceState()) {
    _subscription = _client.watchDownloads().listen(_applyDownload);
  }

  final RestickClient _client;
  late final StreamSubscription<PackDownloadState> _subscription;

  Future<void> load({String? tag}) async {
    emit(state.copyWith(isLoading: true, clearFailure: true));
    try {
      final List<StickerPack> packs = await _client.listMarketplace(tag: tag);
      if (isClosed) {
        return;
      }
      emit(state.copyWith(packs: packs, isLoading: false));
    } on Failure catch (failure) {
      if (isClosed) {
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

  void _applyDownload(PackDownloadState download) {
    final List<StickerPack> next = state.packs.map((StickerPack pack) {
      if (pack.id != download.packId) {
        return pack;
      }
      return pack.copyWith(
        downloadStatus: download.status,
        downloadProgress: download.progress,
      );
    }).toList();
    emit(state.copyWith(packs: next, clearFailure: true));
  }

  @override
  Future<void> close() {
    return _subscription.cancel().whenComplete(super.close);
  }
}
