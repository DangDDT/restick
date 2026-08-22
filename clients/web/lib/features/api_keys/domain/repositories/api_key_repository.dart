import "package:restick_web/features/api_keys/domain/entities/api_key_entity.dart";

abstract class ApiKeyRepository {
  Future<List<ApiKeyEntity>> listMine({bool forceRefresh = false});
  Future<List<ApiKeyEntity>> listAll({bool forceRefresh = false});
  Future<ApiKeyEntity> createMine({required String name, int? expiresInDays});
  Future<ApiKeyEntity> rotateMine({required String id, int? expiresInDays});
  Future<void> deleteMine(String id);
  Future<void> deleteAdmin(String id);
}
