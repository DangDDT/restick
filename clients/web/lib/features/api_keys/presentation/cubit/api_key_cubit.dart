import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/api_keys/domain/entities/api_key_entity.dart";
import "package:restick_web/features/api_keys/domain/repositories/api_key_repository.dart";

class ApiKeyState extends Equatable {
  const ApiKeyState({
    this.personalKeys = const <ApiKeyEntity>[],
    this.allKeys = const <ApiKeyEntity>[],
    this.isRefreshing = false,
    this.hasLoaded = false,
    this.errorMessage,
    this.justCreatedKey,
  });

  final List<ApiKeyEntity> personalKeys;
  final List<ApiKeyEntity> allKeys;
  final bool isRefreshing;
  final bool hasLoaded;
  final String? errorMessage;
  final ApiKeyEntity? justCreatedKey;

  ApiKeyState copyWith({
    List<ApiKeyEntity>? personalKeys,
    List<ApiKeyEntity>? allKeys,
    bool? isRefreshing,
    bool? hasLoaded,
    String? errorMessage,
    ApiKeyEntity? justCreatedKey,
    bool clearError = false,
    bool clearJustCreated = false,
  }) {
    return ApiKeyState(
      personalKeys: personalKeys ?? this.personalKeys,
      allKeys: allKeys ?? this.allKeys,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      hasLoaded: hasLoaded ?? this.hasLoaded,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      justCreatedKey: clearJustCreated
          ? null
          : (justCreatedKey ?? this.justCreatedKey),
    );
  }

  @override
  List<Object?> get props => <Object?>[
        personalKeys,
        allKeys,
        isRefreshing,
        hasLoaded,
        errorMessage,
        justCreatedKey,
      ];
}

class ApiKeyCubit extends Cubit<ApiKeyState> {
  ApiKeyCubit({required ApiKeyRepository apiKeyRepository})
      : _apiKeyRepository = apiKeyRepository,
        super(const ApiKeyState());

  final ApiKeyRepository _apiKeyRepository;

  Future<void> load({bool isAdmin = false, bool forceRefresh = false}) async {
    emit(state.copyWith(isRefreshing: true, clearError: true));
    try {
      final List<ApiKeyEntity> mine =
          await _apiKeyRepository.listMine(forceRefresh: forceRefresh);
      List<ApiKeyEntity> all = const <ApiKeyEntity>[];
      if (isAdmin) {
        all = await _apiKeyRepository.listAll(forceRefresh: forceRefresh);
      }
      emit(
        state.copyWith(
          personalKeys: mine,
          allKeys: all,
          isRefreshing: false,
          hasLoaded: true,
          clearError: true,
        ),
      );
    } on Failure catch (f) {
      emit(state.copyWith(isRefreshing: false, errorMessage: f.message));
    }
  }

  Future<ApiKeyEntity?> createKey({
    required String name,
    int? expiresInDays,
    bool isAdmin = false,
  }) async {
    try {
      final ApiKeyEntity created = await _apiKeyRepository.createMine(
        name: name,
        expiresInDays: expiresInDays,
      );
      await load(isAdmin: isAdmin, forceRefresh: true);
      return created;
    } on Failure catch (f) {
      emit(state.copyWith(errorMessage: f.message));
      return null;
    }
  }

  Future<ApiKeyEntity?> rotateKey({
    required String id,
    int? expiresInDays,
    bool isAdmin = false,
  }) async {
    try {
      final ApiKeyEntity rotated = await _apiKeyRepository.rotateMine(
        id: id,
        expiresInDays: expiresInDays,
      );
      await load(isAdmin: isAdmin, forceRefresh: true);
      return rotated;
    } on Failure catch (f) {
      emit(state.copyWith(errorMessage: f.message));
      return null;
    }
  }

  Future<void> deleteKey({
    required String id,
    bool isAdmin = false,
    bool fromAdminView = false,
  }) async {
    try {
      if (fromAdminView && isAdmin) {
        await _apiKeyRepository.deleteAdmin(id);
      } else {
        await _apiKeyRepository.deleteMine(id);
      }
      await load(isAdmin: isAdmin, forceRefresh: true);
    } on Failure catch (f) {
      emit(state.copyWith(errorMessage: f.message));
    }
  }
}
