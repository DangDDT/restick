import "package:dio/dio.dart";
import "package:restick_web/features/packs/data/models/pack_dto.dart";
import "package:retrofit/retrofit.dart";

part "pack_api.g.dart";

@RestApi()
abstract class PackApi {
  factory PackApi(Dio dio, {String? baseUrl}) = _PackApi;

  @POST("/api/v1/packs")
  Future<CreatePackResponseDto> createPack(@Body() Map<String, dynamic> body);

  @GET("/api/v1/me/packs")
  Future<PackListResponseDto> listMine();

  @GET("/api/v1/admin/packs")
  Future<PackListResponseDto> listQueue();

  @GET("/api/v1/admin/published")
  Future<PackListResponseDto> listPublished();

  @GET("/api/v1/packs/{id}")
  Future<PackDetailDto> getMine(@Path("id") String id);

  @GET("/api/v1/admin/packs/{id}")
  Future<PackDetailDto> getAdmin(@Path("id") String id);

  @POST("/api/v1/packs/{id}/submit")
  Future<void> submit(@Path("id") String id);

  @POST("/api/v1/packs/{id}/confirm")
  Future<void> confirm(@Path("id") String id);

  @POST("/api/v1/admin/packs/{id}/approve")
  Future<void> approve(@Path("id") String id);

  @POST("/api/v1/admin/packs/{id}/reject")
  Future<void> reject(
    @Path("id") String id,
    @Body() Map<String, dynamic> body,
  );

  @POST("/api/v1/admin/packs/{id}/revoke")
  Future<void> revoke(
    @Path("id") String id,
    @Body() Map<String, dynamic> body,
  );

  @DELETE("/api/v1/packs/{id}")
  Future<void> deleteRejected(@Path("id") String id);

  @POST("/api/v1/packs/{id}/revoke")
  Future<void> ownerRevoke(@Path("id") String id);

  @PUT("/api/v1/packs/{id}")
  Future<PackDetailDto> updatePack(
    @Path("id") String id,
    @Body() Map<String, dynamic> body,
  );

  @PUT("/api/v1/packs/{id}/stickers")
  Future<PackDetailDto> updateStickersMetadata(
    @Path("id") String id,
    @Body() Map<String, dynamic> body,
  );

  @GET("/api/v1/admin/analytics/rejections")
  Future<RejectionAnalyticsResponseDto> getRejectionAnalytics();
}
