import "package:dio/dio.dart";
import "package:restick/src/data/dto/pack_dto.dart";
import "package:retrofit/retrofit.dart";

part "restick_api.g.dart";

@RestApi()
abstract class RestickApi {
  factory RestickApi(Dio dio, {String? baseUrl}) = _RestickApi;

  @GET("/api/v1/packs")
  Future<PackListResponseDto> listPacks({@Query("tag") String? tag});

  @GET("/api/v1/packs/{id}")
  Future<PackSummaryDto> getPack(@Path("id") String id);

  @GET("/api/v1/packs/{id}/bundle")
  Future<BundleMetaDto> getBundle(@Path("id") String id);

  @GET("/api/v1/tombstones")
  Future<TombstoneListResponseDto> listTombstones({
    @Query("since") required String since,
  });
}
