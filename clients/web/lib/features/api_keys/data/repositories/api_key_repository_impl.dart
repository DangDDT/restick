import "package:restick_web/core/error/dio_exception_mapper.dart";
import "package:restick_web/features/api_keys/data/datasources/api_keys_api.dart";
import "package:restick_web/features/api_keys/data/models/api_key_dto.dart";
import "package:restick_web/features/api_keys/domain/entities/api_key_entity.dart";
import "package:restick_web/features/api_keys/domain/repositories/api_key_repository.dart";

class ApiKeyRepositoryImpl implements ApiKeyRepository {
  ApiKeyRepositoryImpl({required ApiKeysApi apiKeysApi})
      : _apiKeysApi = apiKeysApi;

  final ApiKeysApi _apiKeysApi;

  @override
  Future<List<ApiKeyEntity>> listMine({bool forceRefresh = false}) async {
    try {
      final ApiKeyListResponseDto res = await _apiKeysApi.listMine();
      return res.items.map((ApiKeyDto dto) => dto.toEntity()).toList();
    } on Exception catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<List<ApiKeyEntity>> listAll({bool forceRefresh = false}) async {
    try {
      final ApiKeyListResponseDto res = await _apiKeysApi.listAll();
      return res.items.map((ApiKeyDto dto) => dto.toEntity()).toList();
    } on Exception catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<ApiKeyEntity> createMine({required String name, int? expiresInDays}) async {
    try {
      final Map<String, dynamic> body = <String, dynamic>{
        "name": name,
        if (expiresInDays != null && expiresInDays > 0)
          "expires_in_days": expiresInDays,
      };
      final ApiKeyDto dto = await _apiKeysApi.createMine(body);
      return dto.toEntity();
    } on Exception catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<ApiKeyEntity> rotateMine({required String id, int? expiresInDays}) async {
    try {
      final Map<String, dynamic> body = <String, dynamic>{
        if (expiresInDays != null && expiresInDays > 0)
          "expires_in_days": expiresInDays,
      };
      final ApiKeyDto dto = await _apiKeysApi.rotateMine(id, body);
      return dto.toEntity();
    } on Exception catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<void> deleteMine(String id) async {
    try {
      await _apiKeysApi.deleteMine(id);
    } on Exception catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<void> deleteAdmin(String id) async {
    try {
      await _apiKeysApi.deleteAdmin(id);
    } on Exception catch (e) {
      throw mapDioException(e);
    }
  }
}
