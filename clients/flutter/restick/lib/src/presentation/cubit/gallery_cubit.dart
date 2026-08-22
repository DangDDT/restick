import "dart:async";

import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/error/failure.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";

class GalleryState extends Equatable {
  const GalleryState({
    this.packs = const <StickerPack>[],
    this.stickers = const <StickerAsset>[],
    this.selectedPackId,
    this.isLoading = false,
    this.failure,
  });

  final List<StickerPack> packs;
  final List<StickerAsset> stickers;
  final String? selectedPackId;
  final bool isLoading;
  final Failure? failure;

  StickerPack? get selectedPack {
    for (final StickerPack pack in packs) {
      if (pack.id == selectedPackId) {
        return pack;
      }
    }
    return null;
  }

  GalleryState copyWith({
    List<StickerPack>? packs,
    List<StickerAsset>? stickers,
    String? selectedPackId,
    bool? isLoading,
    Failure? failure,
    bool clearFailure = false,
  }) {
    return GalleryState(
      packs: packs ?? this.packs,
      stickers: stickers ?? this.stickers,
      selectedPackId: selectedPackId ?? this.selectedPackId,
      isLoading: isLoading ?? this.isLoading,
      failure: clearFailure ? null : failure ?? this.failure,
    );
  }

  @override
  List<Object?> get props => <Object?>[
        packs,
        stickers,
        selectedPackId,
        isLoading,
        failure,
      ];
}

class GalleryCubit extends Cubit<GalleryState> {
  GalleryCubit({required RestickClient client})
      : _client = client,
        super(const GalleryState()) {
    _subscription = _client.watchDownloads().listen(_onDownload);
  }

  final RestickClient _client;
  late final StreamSubscription<PackDownloadState> _subscription;

  Future<void> load() async {
    emit(state.copyWith(isLoading: true, clearFailure: true));
    try {
      final List<StickerPack> packs = await _client.listGallery();
      if (isClosed) {
        return;
      }
      if (packs.isEmpty) {
        emit(
          state.copyWith(
            packs: packs,
            stickers: <StickerAsset>[],
            isLoading: false,
          ),
        );
        return;
      }
      final String? lastId = await _client.lastSelectedPackId();
      final String selectedId = packs.any((StickerPack pack) => pack.id == lastId)
          ? lastId!
          : packs.first.id;
      await selectPack(selectedId, packs: packs);
    } on Failure catch (failure) {
      if (isClosed) {
        return;
      }
      emit(state.copyWith(isLoading: false, failure: failure));
    }
  }

  Future<void> selectPack(String packId, {List<StickerPack>? packs}) async {
    final List<StickerPack> catalog = packs ?? state.packs;
    emit(
      state.copyWith(
        packs: catalog,
        selectedPackId: packId,
        isLoading: true,
        clearFailure: true,
      ),
    );
    await _client.selectPack(packId);
    try {
      final List<StickerAsset> stickers = await _client.getStickers(packId);
      if (isClosed) {
        return;
      }
      emit(
        state.copyWith(
          stickers: stickers,
          isLoading: false,
        ),
      );
    } on Failure catch (failure) {
      if (isClosed) {
        return;
      }
      emit(state.copyWith(isLoading: false, failure: failure));
    }
  }

  Future<void> _onDownload(PackDownloadState download) async {
    if (download.status == PackDownloadStatus.downloaded ||
        download.status == PackDownloadStatus.revoked ||
        download.status == PackDownloadStatus.remote) {
      await load();
    }
  }

  @override
  Future<void> close() {
    return _subscription.cancel().whenComplete(super.close);
  }
}
