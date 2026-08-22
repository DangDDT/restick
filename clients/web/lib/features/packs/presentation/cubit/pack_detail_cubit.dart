import "dart:async";

import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class PackDetailState extends Equatable {
  const PackDetailState({
    this.detail,
    this.isRefreshing = false,
    this.errorMessage,
  });

  final PackDetail? detail;
  final bool isRefreshing;
  final String? errorMessage;

  PackDetailState copyWith({
    PackDetail? detail,
    bool? isRefreshing,
    String? errorMessage,
    bool clearError = false,
  }) {
    return PackDetailState(
      detail: detail ?? this.detail,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
    );
  }

  @override
  List<Object?> get props => <Object?>[detail, isRefreshing, errorMessage];
}

class PackDetailCubit extends Cubit<PackDetailState> {
  PackDetailCubit({
    required PackRepository packRepository,
    required this.packId,
  })  : _packRepository = packRepository,
        super(const PackDetailState());

  final PackRepository _packRepository;
  final String packId;
  Timer? _timer;

  Future<void> load({bool forceRefresh = true}) async {
    emit(state.copyWith(isRefreshing: true, clearError: true));
    try {
      final PackDetail detail = await _packRepository.getMine(
        packId,
        forceRefresh: forceRefresh,
      );
      emit(
        state.copyWith(
          detail: detail,
          isRefreshing: false,
          clearError: true,
        ),
      );
      _timer?.cancel();
      if (detail.summary.isProcessing) {
        _timer = Timer(const Duration(seconds: 2), load);
      }
    } on Failure catch (failure) {
      emit(
        state.copyWith(isRefreshing: false, errorMessage: failure.message),
      );
    }
  }

  Future<void> confirmPublish() async {
    try {
      await _packRepository.confirm(packId);
      await load();
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }

  Future<void> updatePackInfo({
    required String name,
    required String slug,
    required List<String> tags,
  }) async {
    try {
      final PackDetail updated = await _packRepository.updatePack(
        id: packId,
        name: name,
        slug: slug,
        tags: tags,
      );
      emit(state.copyWith(detail: updated, clearError: true));
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }

  Future<void> updateStickerMetadata({
    required String file,
    required String emojiHint,
    required List<String> keywords,
  }) async {
    try {
      final PackDetail updated = await _packRepository.updateStickersMetadata(
        id: packId,
        items: <({String emojiHint, String file, List<String> keywords})>[
          (file: file, emojiHint: emojiHint, keywords: keywords),
        ],
      );
      emit(state.copyWith(detail: updated, clearError: true));
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }

  @override
  Future<void> close() {
    _timer?.cancel();
    return super.close();
  }
}
