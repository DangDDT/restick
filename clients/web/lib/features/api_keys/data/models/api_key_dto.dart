import "package:json_annotation/json_annotation.dart";
import "package:restick_web/features/api_keys/domain/entities/api_key_entity.dart";

part "api_key_dto.g.dart";

@JsonSerializable()
class ApiKeyDto {
  const ApiKeyDto({
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

  factory ApiKeyDto.fromJson(Map<String, dynamic> json) =>
      _$ApiKeyDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ApiKeyDtoToJson(this);

  final String id;
  @JsonKey(name: "user_id")
  final String? userId;
  @JsonKey(name: "user_email")
  final String? userEmail;
  final String name;
  @JsonKey(name: "key_prefix")
  final String keyPrefix;
  @JsonKey(name: "is_active")
  final bool isActive;
  @JsonKey(name: "expires_at")
  final String? expiresAt;
  @JsonKey(name: "last_used_at")
  final String? lastUsedAt;
  @JsonKey(name: "created_at")
  final String createdAt;
  @JsonKey(name: "raw_key")
  final String? rawKey;

  ApiKeyEntity toEntity() {
    return ApiKeyEntity(
      id: id,
      userId: userId,
      userEmail: userEmail,
      name: name,
      keyPrefix: keyPrefix,
      isActive: isActive,
      expiresAt: expiresAt != null ? DateTime.tryParse(expiresAt!) : null,
      lastUsedAt: lastUsedAt != null ? DateTime.tryParse(lastUsedAt!) : null,
      createdAt: DateTime.tryParse(createdAt) ?? DateTime.now(),
      rawKey: rawKey,
    );
  }
}

@JsonSerializable()
class ApiKeyListResponseDto {
  const ApiKeyListResponseDto({this.items = const <ApiKeyDto>[]});

  factory ApiKeyListResponseDto.fromJson(Map<String, dynamic> json) =>
      _$ApiKeyListResponseDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ApiKeyListResponseDtoToJson(this);

  final List<ApiKeyDto> items;
}
