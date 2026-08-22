import "package:dio/dio.dart";
import "package:restick/src/data/dto/pack_dto.dart";

class RestickApi {
  const RestickApi(this._dio, {String? baseUrl}) : _baseUrl = baseUrl;

  final Dio _dio;
  final String? _baseUrl;

  Options? _requestOptions() {
    if (_baseUrl == null) return null;
    return Options(extra: <String, dynamic>{"baseUrl": _baseUrl});
  }

  Future<PackListResponseDto> listPacks({String? tag}) async {
    final Response<dynamic> response = await _dio.get<dynamic>(
      "/api/v1/packs",
      queryParameters: tag != null ? <String, dynamic>{"tag": tag} : null,
      options: _requestOptions(),
    );
    final Map<String, dynamic> data =
        (response.data as Map).cast<String, dynamic>();
    return PackListResponseDto.fromJson(data);
  }

  Future<PackSummaryDto> getPack(String id) async {
    final Response<dynamic> response = await _dio.get<dynamic>(
      "/api/v1/packs/$id",
      options: _requestOptions(),
    );
    final Map<String, dynamic> data =
        (response.data as Map).cast<String, dynamic>();
    return PackSummaryDto.fromJson(data);
  }

  Future<BundleMetaDto> getBundle(String id) async {
    final Response<dynamic> response = await _dio.get<dynamic>(
      "/api/v1/packs/$id/bundle",
      options: _requestOptions(),
    );
    final Map<String, dynamic> data =
        (response.data as Map).cast<String, dynamic>();
    return BundleMetaDto.fromJson(data);
  }

  Future<TombstoneListResponseDto> listTombstones({
    required String since,
  }) async {
    final Response<dynamic> response = await _dio.get<dynamic>(
      "/api/v1/tombstones",
      queryParameters: <String, dynamic>{"since": since},
      options: _requestOptions(),
    );
    final Map<String, dynamic> data =
        (response.data as Map).cast<String, dynamic>();
    return TombstoneListResponseDto.fromJson(data);
  }
}
