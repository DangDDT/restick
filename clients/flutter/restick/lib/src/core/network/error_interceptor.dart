import "package:dio/dio.dart";
import "package:restick/src/core/error/dio_exception_mapper.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";

class ErrorInterceptor extends Interceptor {
  ErrorInterceptor({required RestickErrorCubit errorCubit})
      : _errorCubit = errorCubit;

  final RestickErrorCubit _errorCubit;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    _errorCubit.report(mapDioException(err));
    handler.next(err);
  }
}
