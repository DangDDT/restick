import "package:dio/dio.dart";
import "package:restick_web/features/api_keys/data/models/api_key_dto.dart";
import "package:retrofit/retrofit.dart";

part "api_keys_api.g.dart";

@RestApi()
abstract class ApiKeysApi {
  factory ApiKeysApi(Dio dio, {String? baseUrl}) = _ApiKeysApi;

  @GET("/api/v1/me/api-keys")
  Future<ApiKeyListResponseDto> listMine();

  @POST("/api/v1/me/api-keys")
  Future<ApiKeyDto> createMine(@Body() Map<String, dynamic> body);

  @POST("/api/v1/me/api-keys/{id}/rotate")
  Future<ApiKeyDto> rotateMine(
    @Path("id") String id,
    @Body() Map<String, dynamic> body,
  );

  @DELETE("/api/v1/me/api-keys/{id}")
  Future<void> deleteMine(@Path("id") String id);

  @GET("/api/v1/admin/api-keys")
  Future<ApiKeyListResponseDto> listAll();

  @DELETE("/api/v1/admin/api-keys/{id}")
  Future<void> deleteAdmin(@Path("id") String id);
}
