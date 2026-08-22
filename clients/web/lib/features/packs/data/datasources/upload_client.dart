import "package:dio/dio.dart";
import "package:restick_web/core/error/dio_exception_mapper.dart";

class UploadClient {
  UploadClient({required Dio ftpDio}) : _ftpDio = ftpDio;

  final Dio _ftpDio;

  Future<void> putObject({
    required String key,
    required List<int> bytes,
    required String contentType,
  }) async {
    try {
      await _ftpDio.put<void>(
        "/v1/objects/$key",
        data: bytes,
        options: Options(
          headers: <String, Object?>{
            Headers.contentTypeHeader: contentType,
            Headers.contentLengthHeader: bytes.length,
          },
        ),
      );
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }
}
