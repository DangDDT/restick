import "package:dio/dio.dart";
import "package:restick/src/core/config/restick_config.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";
import "package:restick/src/core/network/dio_factory.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/core/storage/prefs_storage.dart";
import "package:restick/src/core/storage/secure_storage.dart";
import "package:restick/src/data/datasources/restick_api.dart";
import "package:restick/src/data/local/app_database.dart";
import "package:restick/src/data/local/pack_downloader.dart";
import "package:restick/src/data/local/pack_file_store.dart";
import "package:restick/src/data/repositories/pack_repository_impl.dart";
import "package:shared_preferences/shared_preferences.dart";

abstract final class Restick {
  static const String sdkVersion = RestickClient.sdkVersion;

  static Future<RestickClient> open(RestickConfig config) async {
    final SharedPreferences preferences = await SharedPreferences.getInstance();
    final RestickSecureStorage secureStorage = RestickSecureStorage();
    if (config.persistApiKey) {
      await secureStorage.writeApiKey(config.apiKey);
    }
    final RestickPrefsStorage prefs = RestickPrefsStorage(
      preferences: preferences,
    );
    final RestickLoadingCubit loadingCubit = RestickLoadingCubit();
    final RestickErrorCubit errorCubit = RestickErrorCubit(
      onFailure: config.onError,
    );
    final Dio catalogDio = createCatalogDio(
      config: config,
      loadingCubit: loadingCubit,
      errorCubit: errorCubit,
    );
    final Dio downloadDio = createDownloadDio(config: config);
    final RestickDatabase database =
        await RestickDatabase.open(config.databaseExecutor);
    final PackFileStore fileStore = PackFileStore();
    await fileStore.open();
    final PackRepositoryImpl repository = PackRepositoryImpl(
      api: RestickApi(catalogDio),
      downloadDio: downloadDio,
      database: database,
      fileStore: fileStore,
      downloader: PackDownloader(),
      prefs: prefs,
      apiBase: config.apiBase,
      errorCubit: errorCubit,
    );
    final RestickClient client = RestickClient(
      repository: repository,
      config: config,
      loadingCubit: loadingCubit,
      errorCubit: errorCubit,
      dios: <Dio>[catalogDio, downloadDio],
    );
    await client.syncRevocations();
    client.startPeriodicAutoSync();
    return client;
  }
}
