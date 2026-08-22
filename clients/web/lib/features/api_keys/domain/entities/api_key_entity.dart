import "package:equatable/equatable.dart";

class ApiKeyEntity extends Equatable {
  const ApiKeyEntity({
    required this.id,
    required this.name,
    required this.keyPrefix,
    required this.isActive,
    required this.createdAt,
    this.userId,
    this.userEmail,
    this.expiresAt,
    this.lastUsedAt,
    this.rawKey,
  });

  final String id;
  final String? userId;
  final String? userEmail;
  final String name;
  final String keyPrefix;
  final bool isActive;
  final DateTime? expiresAt;
  final DateTime? lastUsedAt;
  final DateTime createdAt;
  final String? rawKey;

  bool get isExpired {
    if (expiresAt == null) {
      return false;
    }
    return DateTime.now().toUtc().isAfter(expiresAt!);
  }

  @override
  List<Object?> get props => <Object?>[
        id,
        userId,
        userEmail,
        name,
        keyPrefix,
        isActive,
        expiresAt,
        lastUsedAt,
        createdAt,
        rawKey,
      ];
}
