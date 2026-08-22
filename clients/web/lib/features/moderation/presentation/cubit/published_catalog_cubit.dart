import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class PublishedCatalogState extends Equatable {
  const PublishedCatalogState({
    this.items = const <PackSummary>[],
    this.isRefreshing = false,
    this.hasLoaded = false,
    this.errorMessage,
  });

  final List<PackSummary> items;
  final bool isRefreshing;
  final bool hasLoaded;
  final String? errorMessage;

  PublishedCatalogState copyWith({
    List<PackSummary>? items,
    bool? isRefreshing,
    bool? hasLoaded,
    String? errorMessage,
    bool clearError = false,
  }) {
    return PublishedCatalogState(
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

class PublishedCatalogCubit extends Cubit<PublishedCatalogState> {
  PublishedCatalogCubit({required PackRepository packRepository})
      : _packRepository = packRepository,
        super(const PublishedCatalogState());

  final PackRepository _packRepository;

  Future<void> load({bool forceRefresh = false}) async {
    emit(state.copyWith(isRefreshing: true, clearError: true));
    try {
      final List<PackSummary> items = await _packRepository.listPublished(
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

  Future<void> revoke({
    required String id,
    required String reasonCode,
    required String reasonText,
  }) async {
    try {
      await _packRepository.revoke(
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
