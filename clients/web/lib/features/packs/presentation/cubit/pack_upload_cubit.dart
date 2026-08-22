import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class PackUploadState extends Equatable {
  const PackUploadState({
    this.isSubmitting = false,
    this.createdPackId,
    this.errorMessage,
  });

  final bool isSubmitting;
  final String? createdPackId;
  final String? errorMessage;

  PackUploadState copyWith({
    bool? isSubmitting,
    String? createdPackId,
    String? errorMessage,
    bool clearError = false,
    bool clearCreated = false,
  }) {
    return PackUploadState(
      isSubmitting: isSubmitting ?? this.isSubmitting,
      createdPackId: clearCreated ? null : (createdPackId ?? this.createdPackId),
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
    );
  }

  @override
  List<Object?> get props =>
      <Object?>[isSubmitting, createdPackId, errorMessage];
}

class PackUploadCubit extends Cubit<PackUploadState> {
  PackUploadCubit({required PackRepository packRepository})
      : _packRepository = packRepository,
        super(const PackUploadState());

  final PackRepository _packRepository;

  Future<void> submit({
    required String name,
    required String slug,
    required String author,
    required List<String> tags,
    required String sourceKind,
    required List<UploadFile> files,
  }) async {
    emit(state.copyWith(isSubmitting: true, clearError: true, clearCreated: true));
    try {
      final String id = await _packRepository.createAndUpload(
        name: name,
        slug: slug,
        author: author,
        tags: tags,
        sourceKind: sourceKind,
        files: files,
      );
      emit(
        state.copyWith(
          isSubmitting: false,
          createdPackId: id,
          clearError: true,
        ),
      );
    } on Failure catch (failure) {
      emit(
        state.copyWith(isSubmitting: false, errorMessage: failure.message),
      );
      rethrow;
    }
  }
}
