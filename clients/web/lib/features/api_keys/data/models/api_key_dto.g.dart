// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'api_key_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ApiKeyDto _$ApiKeyDtoFromJson(Map<String, dynamic> json) => ApiKeyDto(
  id: json['id'] as String,
  name: json['name'] as String,
  keyPrefix: json['key_prefix'] as String,
  isActive: json['is_active'] as bool,
  createdAt: json['created_at'] as String,
  userId: json['user_id'] as String?,
  userEmail: json['user_email'] as String?,
  expiresAt: json['expires_at'] as String?,
  lastUsedAt: json['last_used_at'] as String?,
  rawKey: json['raw_key'] as String?,
);

Map<String, dynamic> _$ApiKeyDtoToJson(ApiKeyDto instance) => <String, dynamic>{
  'id': instance.id,
  'user_id': instance.userId,
  'user_email': instance.userEmail,
  'name': instance.name,
  'key_prefix': instance.keyPrefix,
  'is_active': instance.isActive,
  'expires_at': instance.expiresAt,
  'last_used_at': instance.lastUsedAt,
  'created_at': instance.createdAt,
  'raw_key': instance.rawKey,
};

ApiKeyListResponseDto _$ApiKeyListResponseDtoFromJson(
  Map<String, dynamic> json,
) => ApiKeyListResponseDto(
  items:
      (json['items'] as List<dynamic>?)
          ?.map((e) => ApiKeyDto.fromJson(e as Map<String, dynamic>))
          .toList() ??
      const <ApiKeyDto>[],
);

Map<String, dynamic> _$ApiKeyListResponseDtoToJson(
  ApiKeyListResponseDto instance,
) => <String, dynamic>{'items': instance.items.map((e) => e.toJson()).toList()};
