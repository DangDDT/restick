// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'pack_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

PackSummaryDto _$PackSummaryDtoFromJson(Map<String, dynamic> json) =>
    PackSummaryDto(
      id: json['id'] as String,
      slug: json['slug'] as String,
      name: json['name'] as String,
      author: json['author'] as String,
      tags: (json['tags'] as List<dynamic>).map((e) => e as String).toList(),
      stickerCount: (json['sticker_count'] as num).toInt(),
      version: (json['version'] as num).toInt(),
      status: json['status'] as String,
      thumbnailUrl: json['thumbnail_url'] as String? ?? '',
      checksumSha256: json['checksum_sha256'] as String? ?? '',
    );

Map<String, dynamic> _$PackSummaryDtoToJson(PackSummaryDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'slug': instance.slug,
      'name': instance.name,
      'author': instance.author,
      'tags': instance.tags,
      'sticker_count': instance.stickerCount,
      'version': instance.version,
      'status': instance.status,
      'thumbnail_url': instance.thumbnailUrl,
      'checksum_sha256': instance.checksumSha256,
    };

StickerPreviewDto _$StickerPreviewDtoFromJson(Map<String, dynamic> json) =>
    StickerPreviewDto(
      file: json['file'] as String,
      emojiHint: json['emoji_hint'] as String? ?? '',
      width: (json['width'] as num).toInt(),
      height: (json['height'] as num).toInt(),
      fileUrl: json['file_url'] as String? ?? '',
      thumbUrl: json['thumb_url'] as String? ?? '',
      keywords:
          (json['keywords'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          [],
    );

Map<String, dynamic> _$StickerPreviewDtoToJson(StickerPreviewDto instance) =>
    <String, dynamic>{
      'file': instance.file,
      'emoji_hint': instance.emojiHint,
      'keywords': instance.keywords,
      'width': instance.width,
      'height': instance.height,
      'file_url': instance.fileUrl,
      'thumb_url': instance.thumbUrl,
    };

PackDetailDto _$PackDetailDtoFromJson(Map<String, dynamic> json) =>
    PackDetailDto(
      summary: PackSummaryDto.fromJson(json['summary'] as Map<String, dynamic>),
      stickers:
          (json['stickers'] as List<dynamic>)
              .map((e) => StickerPreviewDto.fromJson(e as Map<String, dynamic>))
              .toList(),
      rejectReason: json['rejectReason'] as String,
      uploadPrefix: json['uploadPrefix'] as String,
    );

Map<String, dynamic> _$PackDetailDtoToJson(PackDetailDto instance) =>
    <String, dynamic>{
      'summary': instance.summary.toJson(),
      'stickers': instance.stickers.map((e) => e.toJson()).toList(),
      'rejectReason': instance.rejectReason,
      'uploadPrefix': instance.uploadPrefix,
    };

PackListResponseDto _$PackListResponseDtoFromJson(Map<String, dynamic> json) =>
    PackListResponseDto(
      items:
          (json['items'] as List<dynamic>)
              .map((e) => PackSummaryDto.fromJson(e as Map<String, dynamic>))
              .toList(),
    );

Map<String, dynamic> _$PackListResponseDtoToJson(
  PackListResponseDto instance,
) => <String, dynamic>{'items': instance.items.map((e) => e.toJson()).toList()};

CreatePackResponseDto _$CreatePackResponseDtoFromJson(
  Map<String, dynamic> json,
) => CreatePackResponseDto(
  id: json['id'] as String,
  uploadPrefix: json['upload_prefix'] as String,
);

Map<String, dynamic> _$CreatePackResponseDtoToJson(
  CreatePackResponseDto instance,
) => <String, dynamic>{
  'id': instance.id,
  'upload_prefix': instance.uploadPrefix,
};

RejectionStatDto _$RejectionStatDtoFromJson(Map<String, dynamic> json) =>
    RejectionStatDto(
      reasonCode: json['reason_code'] as String? ?? 'other',
      rejectionCount: (json['rejection_count'] as num?)?.toInt() ?? 0,
      lastRejectedAt: json['last_rejected_at'] as String? ?? '',
    );

Map<String, dynamic> _$RejectionStatDtoToJson(RejectionStatDto instance) =>
    <String, dynamic>{
      'reason_code': instance.reasonCode,
      'rejection_count': instance.rejectionCount,
      'last_rejected_at': instance.lastRejectedAt,
    };

RejectionAnalyticsResponseDto _$RejectionAnalyticsResponseDtoFromJson(
  Map<String, dynamic> json,
) => RejectionAnalyticsResponseDto(
  analytics:
      (json['analytics'] as List<dynamic>)
          .map((e) => RejectionStatDto.fromJson(e as Map<String, dynamic>))
          .toList(),
);

Map<String, dynamic> _$RejectionAnalyticsResponseDtoToJson(
  RejectionAnalyticsResponseDto instance,
) => <String, dynamic>{
  'analytics': instance.analytics.map((e) => e.toJson()).toList(),
};
