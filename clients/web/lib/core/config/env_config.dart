import 'package:flutter/foundation.dart' show kIsWeb;

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
    const fromBase = String.fromEnvironment('BASE_URL');
    const fromRestick = String.fromEnvironment('RESTICK_BASE_URL');
    const fromApi = String.fromEnvironment('API_BASE_URL');
    const compiled = fromBase != ''
        ? fromBase
        : (fromRestick != '' ? fromRestick : fromApi);
    final String base;
    if (compiled.isNotEmpty) {
      base = compiled;
    } else if (kIsWeb) {
      base = Uri.base.origin; // same-origin relative to the served domain
    } else {
      base = 'http://localhost';
    }
    return EnvConfig(baseUrl: base);
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
