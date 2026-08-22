import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class ReviewDetailState extends Equatable {
  const ReviewDetailState({
    this.detail,
    this.isRefreshing = false,
    this.errorMessage,
  });

  final PackDetail? detail;
  final bool isRefreshing;
  final String? errorMessage;

  ReviewDetailState copyWith({
    PackDetail? detail,
    bool? isRefreshing,
    String? errorMessage,
    bool clearError = false,
  }) {
    return ReviewDetailState(
      detail: detail ?? this.detail,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
    );
  }

  @override
  List<Object?> get props => <Object?>[detail, isRefreshing, errorMessage];
}

class ReviewDetailCubit extends Cubit<ReviewDetailState> {
  ReviewDetailCubit({
    required PackRepository packRepository,
    required this.packId,
  })  : _packRepository = packRepository,
        super(const ReviewDetailState());

  final PackRepository _packRepository;
  final String packId;

  Future<void> load({bool forceRefresh = true}) async {
    emit(state.copyWith(isRefreshing: true, clearError: true));
    try {
      final PackDetail detail = await _packRepository.getAdmin(
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
    } on Failure catch (failure) {
      emit(
        state.copyWith(isRefreshing: false, errorMessage: failure.message),
      );
    }
  }

  Future<void> approve() async {
    try {
      await _packRepository.approve(packId);
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }

  Future<void> reject({
    required String reasonCode,
    required String reasonText,
  }) async {
    try {
      await _packRepository.reject(
        id: packId,
        reasonCode: reasonCode,
        reasonText: reasonText,
      );
    } on Failure catch (failure) {
      emit(state.copyWith(errorMessage: failure.message));
      rethrow;
    }
  }
}
