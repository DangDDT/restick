/// Runtime configuration for Restick Web.
///
/// Values are injected at build/run time via:
/// `flutter run --dart-define-from-file=configuration.dev.json`
/// `flutter build web --dart-define-from-file=configuration.prod.json`
///
/// Config files live at the project root (not under `assets/`) so they are
/// not published as Flutter asset bundle entries.
class EnvConfig {
  const EnvConfig({
    required this.apiBaseUrl,
    required this.ftpBaseUrl,
  });

  factory EnvConfig.load() {
    return const EnvConfig(
      apiBaseUrl: apiBaseUrlDefine,
      ftpBaseUrl: ftpBaseUrlDefine,
    );
  }

  final String apiBaseUrl;
  final String ftpBaseUrl;

  static const String apiBaseUrlDefine = String.fromEnvironment(
    "API_BASE_URL",
    defaultValue: "http://localhost:8080",
  );

  static const String ftpBaseUrlDefine = String.fromEnvironment(
    "FTP_BASE_URL",
    defaultValue: "http://localhost:8081",
  );
}
