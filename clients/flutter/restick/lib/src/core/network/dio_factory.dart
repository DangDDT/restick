import "package:dio/dio.dart";
import "package:restick/src/core/config/restick_config.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";
import "package:restick/src/core/network/api_key_interceptor.dart";
import "package:restick/src/core/network/error_interceptor.dart";
import "package:restick/src/core/network/loading_interceptor.dart";

Dio createCatalogDio({
  required RestickConfig config,
  required RestickLoadingCubit loadingCubit,
  required RestickErrorCubit errorCubit,
}) {
  final Dio dio = Dio(
    BaseOptions(
      baseUrl: config.apiBaseUrl,
      connectTimeout: const Duration(seconds: 20),
      receiveTimeout: const Duration(seconds: 30),
      headers: <String, Object?>{
        Headers.contentTypeHeader: Headers.jsonContentType,
      },
    ),
  );
  dio.interceptors.addAll(<Interceptor>[
    ApiKeyInterceptor(apiKey: config.apiKey),
    LoadingInterceptor(loadingCubit: loadingCubit),
    ErrorInterceptor(errorCubit: errorCubit),
  ]);
  return dio;
}

Dio createDownloadDio({required RestickConfig config}) {
  final Dio dio = Dio(
    BaseOptions(
      connectTimeout: const Duration(seconds: 30),
      receiveTimeout: const Duration(minutes: 2),
    ),
  );
  dio.interceptors.add(ApiKeyInterceptor(apiKey: config.apiKey));
  return dio;
}
