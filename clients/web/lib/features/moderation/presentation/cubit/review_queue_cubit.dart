import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class ReviewQueueState extends Equatable {
  const ReviewQueueState({
    this.items = const <PackSummary>[],
    this.isRefreshing = false,
    this.hasLoaded = false,
    this.errorMessage,
  });

  final List<PackSummary> items;
  final bool isRefreshing;
  final bool hasLoaded;
  final String? errorMessage;

  ReviewQueueState copyWith({
    List<PackSummary>? items,
    bool? isRefreshing,
    bool? hasLoaded,
    String? errorMessage,
    bool clearError = false,
  }) {
    return ReviewQueueState(
      items: items ?? this.items,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      hasLoaded: hasLoaded ?? this.hasLoaded,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
    );
  }

  @override
  List<Object?> get props =>
      <Object?>[items, isRefreshing, hasLoaded, errorMessage];
}

class ReviewQueueCubit extends Cubit<ReviewQueueState> {
  ReviewQueueCubit({required PackRepository packRepository})
      : _packRepository = packRepository,
        super(const ReviewQueueState());

  final PackRepository _packRepository;

  Future<void> load({bool forceRefresh = false}) async {
    emit(state.copyWith(isRefreshing: true, clearError: true));
    try {
      final List<PackSummary> items = await _packRepository.listQueue(
        forceRefresh: forceRefresh,
      );
      emit(
        state.copyWith(
          items: items,
          isRefreshing: false,
          hasLoaded: true,
          clearError: true,
        ),
      );
    } on Failure catch (failure) {
      emit(
        state.copyWith(isRefreshing: false, errorMessage: failure.message),
      );
    }
  }

  Future<void> approve(String id) async {
    try {
      await _packRepository.approve(id);
      await load(forceRefresh: true);
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }

  Future<void> reject({
    required String id,
    required String reasonCode,
    required String reasonText,
  }) async {
    try {
      await _packRepository.reject(
        id: id,
        reasonCode: reasonCode,
        reasonText: reasonText,
      );
      await load(forceRefresh: true);
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }
}
