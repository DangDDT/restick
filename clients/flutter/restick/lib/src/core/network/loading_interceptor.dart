import "package:dio/dio.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";

class LoadingInterceptor extends Interceptor {
  LoadingInterceptor({required RestickLoadingCubit loadingCubit})
    : _loadingCubit = loadingCubit;

  final RestickLoadingCubit _loadingCubit;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (!_silent(options)) {
      _loadingCubit.start();
    }
    handler.next(options);
  }

  @override
  void onResponse(
    Response<dynamic> response,
    ResponseInterceptorHandler handler,
  ) {
    if (!_silent(response.requestOptions)) {
      _loadingCubit.stop();
    }
    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (!_silent(err.requestOptions)) {
      _loadingCubit.stop();
    }
    handler.next(err);
  }

  bool _silent(RequestOptions options) => options.extra["silent"] == true;
}
