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
    required this.baseUrl,
  });

  factory EnvConfig.load() {
    return const EnvConfig(
      baseUrl: baseUrlDefine,
    );
  }

  /// Single Unified Ingress URL for the entire Restick platform (API, Assets, Docs).
  final String baseUrl;

  /// Backward-compatible accessors for legacy callers.
  String get apiBaseUrl => baseUrl;
  String get ftpBaseUrl => baseUrl;

  static const String baseUrlDefine = String.fromEnvironment(
    "BASE_URL",
    defaultValue: String.fromEnvironment(
      "RESTICK_BASE_URL",
      defaultValue: String.fromEnvironment(
        "API_BASE_URL",
        defaultValue: "http://localhost",
      ),
    ),
  );
}
