import "package:equatable/equatable.dart";
import "package:restick/src/core/error/failure.dart";

class RestickConfig extends Equatable {
  const RestickConfig({
    required this.apiBaseUrl,
    required this.apiKey,
    this.autoSyncInterval = const Duration(minutes: 5),
    this.onError,
    this.persistApiKey = true,
  });

  factory RestickConfig.fromJson(Map<String, Object?> json) {
    final Object? intervalMinutes = json["autoSyncIntervalMinutes"];
    return RestickConfig(
      apiBaseUrl: json["apiBaseUrl"]! as String,
      apiKey: json["apiKey"]! as String,
      autoSyncInterval: intervalMinutes is num
          ? Duration(minutes: intervalMinutes.toInt())
          : const Duration(minutes: 5),
      persistApiKey: json["persistApiKey"] as bool? ?? true,
    );
  }

  factory RestickConfig.fromEnvironment() {
    return const RestickConfig(
      apiBaseUrl: String.fromEnvironment(
        "RESTICK_API_BASE",
        defaultValue: "http://127.0.0.1:8080",
      ),
      apiKey: String.fromEnvironment(
        "RESTICK_API_KEY",
        defaultValue: "dev-api-key",
      ),
    );
  }

  final String apiBaseUrl;
  final String apiKey;
  final Duration autoSyncInterval;
  final void Function(Failure failure)? onError;
  final bool persistApiKey;

  Uri get apiBase => Uri.parse(apiBaseUrl);

  @override
  List<Object?> get props => <Object?>[
        apiBaseUrl,
        apiKey,
        autoSyncInterval,
        persistApiKey,
      ];
}
