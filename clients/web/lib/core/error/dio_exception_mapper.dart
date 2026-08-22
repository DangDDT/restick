import "package:dio/dio.dart";
import "package:restick_web/core/error/failure.dart";

Failure mapDioException(Object error) {
  if (error is! DioException) {
    return UnexpectedFailure(error.toString());
  }
  final Response<dynamic>? response = error.response;
  if (response?.data is Map) {
    final Map<String, Object?> body =
        Map<String, Object?>.from(response!.data as Map);
    final Object? err = body["error"];
    if (err is Map) {
      final Map<String, Object?> errorMap = Map<String, Object?>.from(err);
      final String message =
          (errorMap["message"] as String?) ?? "request failed";
      final int? status = response.statusCode;
      if (status == 401 || status == 403) {
        return AuthFailure(message);
      }
      return ServerFailure(message);
    }
  }
  if (error.type == DioExceptionType.connectionError ||
      error.type == DioExceptionType.connectionTimeout ||
      error.type == DioExceptionType.receiveTimeout ||
      error.type == DioExceptionType.sendTimeout) {
    return const NetworkFailure();
  }
  if (response?.statusCode == 401 || response?.statusCode == 403) {
    return AuthFailure(error.message ?? "Authentication failed");
  }
  return ServerFailure(error.message ?? "Server error");
}
