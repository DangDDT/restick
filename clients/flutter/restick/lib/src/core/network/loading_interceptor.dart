import "package:dio/dio.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";

class LoadingInterceptor extends Interceptor {
  LoadingInterceptor({required RestickLoadingCubit loadingCubit})
      : _loadingCubit = loadingCubit;

  final RestickLoadingCubit _loadingCubit;

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
