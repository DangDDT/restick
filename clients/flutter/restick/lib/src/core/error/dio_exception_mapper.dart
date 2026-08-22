import "package:dio/dio.dart";
import "package:restick/src/core/error/failure.dart";

Failure mapDioException(DioException error, {String packId = ""}) {
  final int? status = error.response?.statusCode;
  if (status == 410) {
    return RevokedFailure(packId);
  }
  if (status == 401 || status == 403) {
    return const AuthFailure();
  }
  if (status != null && status >= 500) {
    return ServerFailure(_readMessage(error) ?? "Server error");
  }
  if (error.type == DioExceptionType.connectionError ||
      error.type == DioExceptionType.connectionTimeout ||
      error.type == DioExceptionType.receiveTimeout ||
      error.type == DioExceptionType.sendTimeout) {
    return const NetworkFailure();
  }
  return ServerFailure(_readMessage(error) ?? "Request failed");
}

String? _readMessage(DioException error) {
  final Object? data = error.response?.data;
  if (data is Map<String, dynamic>) {
    final Object? wrapped = data["error"];
    if (wrapped is Map<String, dynamic>) {
      return wrapped["message"] as String?;
    }
    return data["message"] as String?;
  }
  return error.message;
}
