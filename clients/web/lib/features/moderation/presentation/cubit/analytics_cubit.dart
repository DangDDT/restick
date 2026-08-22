import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class AnalyticsState extends Equatable {
  const AnalyticsState({
    this.items = const <RejectionStat>[],
    this.isRefreshing = false,
    this.hasLoaded = false,
    this.errorMessage,
  });

  final List<RejectionStat> items;
  final bool isRefreshing;
  final bool hasLoaded;
  final String? errorMessage;

  AnalyticsState copyWith({
    List<RejectionStat>? items,
    bool? isRefreshing,
    bool? hasLoaded,
    String? errorMessage,
    bool clearError = false,
  }) {
    return AnalyticsState(
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

class AnalyticsCubit extends Cubit<AnalyticsState> {
  AnalyticsCubit({required PackRepository packRepository})
      : _packRepository = packRepository,
        super(const AnalyticsState());

  final PackRepository _packRepository;

  Future<void> load({bool forceRefresh = false}) async {
    emit(state.copyWith(isRefreshing: true, clearError: true));
    try {
      final List<RejectionStat> items =
          await _packRepository.getRejectionAnalytics(
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
}
