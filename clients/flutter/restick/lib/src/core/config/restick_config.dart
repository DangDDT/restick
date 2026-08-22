import "package:drift/drift.dart";
import "package:equatable/equatable.dart";
import "package:restick/src/core/error/failure.dart";

class RestickConfig extends Equatable {
  const RestickConfig({
    required this.baseUrl,
    required this.apiKey,
    this.autoSyncInterval = const Duration(minutes: 5),
    this.onError,
    this.persistApiKey = true,
    this.databaseExecutor,
  });

  /// Backward-compatible constructor accepting `apiBaseUrl`.
  const RestickConfig.legacy({
    required String apiBaseUrl,
    required String apiKey,
    Duration autoSyncInterval = const Duration(minutes: 5),
    void Function(Failure failure)? onError,
    bool persistApiKey = true,
    QueryExecutor? databaseExecutor,
  }) : this(
          baseUrl: apiBaseUrl,
          apiKey: apiKey,
          autoSyncInterval: autoSyncInterval,
          onError: onError,
          persistApiKey: persistApiKey,
          databaseExecutor: databaseExecutor,
        );

  factory RestickConfig.fromJson(Map<String, Object?> json) {
    final Object? intervalMinutes = json["autoSyncIntervalMinutes"];
    final String resolvedBase =
        (json["baseUrl"] ?? json["apiBaseUrl"])! as String;
    return RestickConfig(
      baseUrl: resolvedBase,
      apiKey: json["apiKey"]! as String,
      autoSyncInterval: intervalMinutes is num
          ? Duration(minutes: intervalMinutes.toInt())
          : const Duration(minutes: 5),
      persistApiKey: json["persistApiKey"] as bool? ?? true,
    );
  }

  factory RestickConfig.fromEnvironment() {
    return const RestickConfig(
      baseUrl: String.fromEnvironment(
        "RESTICK_BASE_URL",
        defaultValue: String.fromEnvironment(
          "RESTICK_API_BASE",
          defaultValue: "http://127.0.0.1",
        ),
      ),
      apiKey: String.fromEnvironment(
        "RESTICK_API_KEY",
        defaultValue: "dev-api-key",
      ),
    );
  }

  /// Single Gateway Ingress URL.
  final String baseUrl;

  /// Backward-compatible alias.
  String get apiBaseUrl => baseUrl;

  final String apiKey;
  final Duration autoSyncInterval;
  final void Function(Failure failure)? onError;
  final bool persistApiKey;

  /// Optional custom QueryExecutor (e.g. injected SQLCipher database opener or custom NativeDatabase).
  final QueryExecutor? databaseExecutor;

  Uri get apiBase => Uri.parse(baseUrl);

  @override
  List<Object?> get props => <Object?>[
        baseUrl,
        apiKey,
        autoSyncInterval,
        persistApiKey,
        databaseExecutor,
      ];
}
