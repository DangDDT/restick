import "package:dio/dio.dart";
import "package:get_it/get_it.dart";
import "package:restick_web/core/config/env_config.dart";
import "package:restick_web/core/database/app_database.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/loading_cubit.dart";
import "package:restick_web/core/network/dio_factory.dart";
import "package:restick_web/core/storage/prefs_storage.dart";
import "package:restick_web/core/storage/secure_storage.dart";
import "package:restick_web/core/theme/theme_cubit.dart";
import "package:restick_web/features/auth/data/datasources/auth_api.dart";
import "package:restick_web/features/auth/data/repositories/auth_repository_impl.dart";
import "package:restick_web/features/auth/domain/repositories/auth_repository.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/features/packs/data/datasources/pack_api.dart";
import "package:restick_web/features/packs/data/datasources/upload_client.dart";
import "package:restick_web/features/packs/data/repositories/pack_repository_impl.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_detail_cubit.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_list_cubit.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_upload_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/analytics_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/published_catalog_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/review_detail_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/review_queue_cubit.dart";
import "package:restick_web/features/api_keys/data/datasources/api_keys_api.dart";
import "package:restick_web/features/api_keys/data/repositories/api_key_repository_impl.dart";
import "package:restick_web/features/api_keys/domain/repositories/api_key_repository.dart";
import "package:restick_web/features/api_keys/presentation/cubit/api_key_cubit.dart";
import "package:restick_web/features/preview/presentation/cubit/preview_cubit.dart";
import "package:shared_preferences/shared_preferences.dart";

final GetIt getIt = GetIt.instance;

Future<void> configureDependencies(EnvConfig config) async {
  final SharedPreferences preferences = await SharedPreferences.getInstance();
  final AppDatabase database = await AppDatabase.open();
  final SecureStorage secureStorage = SecureStorage();
  final PrefsStorage prefsStorage = PrefsStorage(preferences: preferences);
  final LoadingCubit loadingCubit = LoadingCubit();
  final ErrorCubit errorCubit = ErrorCubit();
  final Dio apiDio = createApiDio(
    config: config,
    secureStorage: secureStorage,
    loadingCubit: loadingCubit,
  );
  final Dio ftpDio = createFtpDio(
    config: config,
    secureStorage: secureStorage,
    loadingCubit: loadingCubit,
  );
  final AuthApi authApi = AuthApi(apiDio);
  final PackApi packApi = PackApi(apiDio);
  final UploadClient uploadClient = UploadClient(ftpDio: ftpDio);
  final AuthRepository authRepository = AuthRepositoryImpl(
    authApi: authApi,
    secureStorage: secureStorage,
  );
  final PackRepository packRepository = PackRepositoryImpl(
    packApi: packApi,
    uploadClient: uploadClient,
    database: database,
  );

  final ApiKeysApi apiKeysApi = ApiKeysApi(apiDio);
  final ApiKeyRepository apiKeyRepository = ApiKeyRepositoryImpl(
    apiKeysApi: apiKeysApi,
  );

  getIt
    ..registerSingleton<EnvConfig>(config)
    ..registerSingleton<SharedPreferences>(preferences)
    ..registerSingleton<AppDatabase>(database)
    ..registerSingleton<SecureStorage>(secureStorage)
    ..registerSingleton<PrefsStorage>(prefsStorage)
    ..registerSingleton<LoadingCubit>(loadingCubit)
    ..registerSingleton<ErrorCubit>(errorCubit)
    ..registerSingleton<Dio>(apiDio, instanceName: "api")
    ..registerSingleton<Dio>(ftpDio, instanceName: "ftp")
    ..registerSingleton<AuthApi>(authApi)
    ..registerSingleton<PackApi>(packApi)
    ..registerSingleton<ApiKeysApi>(apiKeysApi)
    ..registerSingleton<UploadClient>(uploadClient)
    ..registerSingleton<AuthRepository>(authRepository)
    ..registerSingleton<PackRepository>(packRepository)
    ..registerSingleton<ApiKeyRepository>(apiKeyRepository)
    ..registerSingleton<ThemeCubit>(ThemeCubit(prefsStorage: prefsStorage))
    ..registerSingleton<LocaleCubit>(LocaleCubit(prefsStorage: prefsStorage))
    ..registerSingleton<AuthCubit>(AuthCubit(authRepository: authRepository))
    ..registerSingleton<PreviewCubit>(PreviewCubit())
    ..registerFactory<ApiKeyCubit>(
      () => ApiKeyCubit(apiKeyRepository: getIt()),
    )
    ..registerFactory<PackListCubit>(
      () => PackListCubit(packRepository: getIt()),
    )
    ..registerFactoryParam<PackDetailCubit, String, void>(
      (String packId, _) => PackDetailCubit(
        packRepository: getIt(),
        packId: packId,
      ),
    )
    ..registerFactory<PackUploadCubit>(
      () => PackUploadCubit(packRepository: getIt()),
    )
    ..registerFactory<ReviewQueueCubit>(
      () => ReviewQueueCubit(packRepository: getIt()),
    )
    ..registerFactoryParam<ReviewDetailCubit, String, void>(
      (String packId, _) => ReviewDetailCubit(
        packRepository: getIt(),
        packId: packId,
      ),
    )
    ..registerFactory<PublishedCatalogCubit>(
      () => PublishedCatalogCubit(packRepository: getIt()),
    )
    ..registerFactory<AnalyticsCubit>(
      () => AnalyticsCubit(packRepository: getIt()),
    );
}
