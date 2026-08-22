import "package:dio/dio.dart";
import "package:restick_web/core/error/loading_cubit.dart";
import "package:restick_web/core/storage/secure_storage.dart";

class AuthInterceptor extends Interceptor {
  AuthInterceptor({required SecureStorage secureStorage})
      : _secureStorage = secureStorage;

  final SecureStorage _secureStorage;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final String? token = await _secureStorage.readToken();
    if (token != null && token.isNotEmpty) {
      options.headers["Authorization"] = "Bearer $token";
    }
    handler.next(options);
  }
}

class LoadingInterceptor extends Interceptor {
  LoadingInterceptor({required LoadingCubit loadingCubit})
      : _loadingCubit = loadingCubit;

  final LoadingCubit _loadingCubit;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    _loadingCubit.start();
    handler.next(options);
  }

  @override
  void onResponse(
    Response<dynamic> response,
    ResponseInterceptorHandler handler,
  ) {
    _loadingCubit.stop();
    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    _loadingCubit.stop();
    handler.next(err);
  }
}
