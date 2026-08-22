import "package:dio/dio.dart";
import "package:restick_web/core/config/env_config.dart";
import "package:restick_web/core/error/loading_cubit.dart";
import "package:restick_web/core/network/auth_interceptor.dart";
import "package:restick_web/core/storage/secure_storage.dart";

Dio createApiDio({
  required EnvConfig config,
  required SecureStorage secureStorage,
  required LoadingCubit loadingCubit,
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
    AuthInterceptor(secureStorage: secureStorage),
    LoadingInterceptor(loadingCubit: loadingCubit),
  ]);
  return dio;
}

Dio createFtpDio({
  required EnvConfig config,
  required SecureStorage secureStorage,
  required LoadingCubit loadingCubit,
}) {
  final Dio dio = Dio(
    BaseOptions(
      baseUrl: config.ftpBaseUrl,
      connectTimeout: const Duration(seconds: 30),
      receiveTimeout: const Duration(minutes: 2),
    ),
  );
  dio.interceptors.addAll(<Interceptor>[
    AuthInterceptor(secureStorage: secureStorage),
    LoadingInterceptor(loadingCubit: loadingCubit),
  ]);
  return dio;
}
