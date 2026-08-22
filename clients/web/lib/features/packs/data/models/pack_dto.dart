import "package:equatable/equatable.dart";
import "package:json_annotation/json_annotation.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";

part "pack_dto.g.dart";

@JsonSerializable()
class PackSummaryDto extends Equatable {
  const PackSummaryDto({
    required this.id,
    required this.slug,
    required this.name,
    required this.author,
    required this.tags,
    required this.stickerCount,
    required this.version,
    required this.status,
    required this.thumbnailUrl,
    required this.checksumSha256,
  });

  factory PackSummaryDto.fromJson(Map<String, dynamic> json) =>
      _$PackSummaryDtoFromJson(json);

  final String id;
  final String slug;
  final String name;
  final String author;
  final List<String> tags;
  @JsonKey(name: "sticker_count")
  final int stickerCount;
  final int version;
  final String status;
  @JsonKey(name: "thumbnail_url", defaultValue: "")
  final String thumbnailUrl;
  @JsonKey(name: "checksum_sha256", defaultValue: "")
  final String checksumSha256;

  Map<String, dynamic> toJson() => _$PackSummaryDtoToJson(this);

  PackSummary toEntity() {
    return PackSummary(
      id: id,
      slug: slug,
      name: name,
      author: author,
      tags: tags,
      stickerCount: stickerCount,
      version: version,
      status: status,
      thumbnailUrl: thumbnailUrl,
      checksumSha256: checksumSha256,
    );
  }

  @override
  List<Object?> get props => <Object?>[
        id,
        slug,
        name,
        author,
        tags,
        stickerCount,
        version,
        status,
        thumbnailUrl,
        checksumSha256,
      ];
}

@JsonSerializable()
class StickerPreviewDto extends Equatable {
  const StickerPreviewDto({
    required this.file,
    required this.emojiHint,
    required this.width,
    required this.height,
    required this.fileUrl,
    required this.thumbUrl,
    this.keywords = const <String>[],
  });

  factory StickerPreviewDto.fromJson(Map<String, dynamic> json) {
    final Object? rawKw = json["keywords"];
    final List<String> kw = rawKw is List
        ? rawKw.map((Object? e) => e.toString()).toList()
        : <String>[];
    return StickerPreviewDto(
      file: (json["file"] as String?) ?? "",
      emojiHint: (json["emoji_hint"] as String?) ?? "",
      keywords: kw,
      width: (json["width"] as int?) ?? 512,
      height: (json["height"] as int?) ?? 512,
      fileUrl: (json["file_url"] as String?) ?? "",
      thumbUrl: (json["thumb_url"] as String?) ?? "",
    );
  }

  final String file;
  @JsonKey(name: "emoji_hint", defaultValue: "")
  final String emojiHint;
  @JsonKey(name: "keywords", defaultValue: <String>[])
  final List<String> keywords;
  final int width;
  final int height;
  @JsonKey(name: "file_url", defaultValue: "")
  final String fileUrl;
  @JsonKey(name: "thumb_url", defaultValue: "")
  final String thumbUrl;

  Map<String, dynamic> toJson() => _$StickerPreviewDtoToJson(this);

  StickerPreview toEntity() {
    return StickerPreview(
      file: file,
      emojiHint: emojiHint,
      keywords: keywords,
      width: width,
      height: height,
      fileUrl: fileUrl,
      thumbUrl: thumbUrl,
    );
  }

  @override
  List<Object?> get props =>
      <Object?>[file, emojiHint, keywords, width, height, fileUrl, thumbUrl];
}

@JsonSerializable()
class PackDetailDto extends Equatable {
  const PackDetailDto({
    required this.summary,
    required this.stickers,
    required this.rejectReason,
    required this.uploadPrefix,
  });

  factory PackDetailDto.fromJson(Map<String, dynamic> json) {
    final PackSummaryDto summary = PackSummaryDto.fromJson(json);
    final Object? rawStickers = json["stickers"];
    final List<StickerPreviewDto> stickers = rawStickers is List
        ? rawStickers
            .map(
              (Object? item) =>
                  StickerPreviewDto.fromJson(Map<String, dynamic>.from(item! as Map)),
            )
            .toList()
        : <StickerPreviewDto>[];
    return PackDetailDto(
      summary: summary,
      stickers: stickers,
      rejectReason: (json["reject_reason"] as String?) ?? "",
      uploadPrefix: (json["upload_prefix"] as String?) ?? "",
    );
  }

  final PackSummaryDto summary;
  final List<StickerPreviewDto> stickers;
  final String rejectReason;
  final String uploadPrefix;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      ...summary.toJson(),
      "stickers": stickers.map((StickerPreviewDto s) => s.toJson()).toList(),
      "reject_reason": rejectReason,
      "upload_prefix": uploadPrefix,
    };
  }

  PackDetail toEntity() {
    return PackDetail(
      summary: summary.toEntity(),
      stickers: stickers.map((StickerPreviewDto s) => s.toEntity()).toList(),
      rejectReason: rejectReason,
      uploadPrefix: uploadPrefix,
    );
  }

  @override
  List<Object?> get props =>
      <Object?>[summary, stickers, rejectReason, uploadPrefix];
}

@JsonSerializable()
class PackListResponseDto extends Equatable {
  const PackListResponseDto({required this.items});

  factory PackListResponseDto.fromJson(Map<String, dynamic> json) =>
      _$PackListResponseDtoFromJson(json);

  final List<PackSummaryDto> items;

  Map<String, dynamic> toJson() => _$PackListResponseDtoToJson(this);

  @override
  List<Object?> get props => <Object?>[items];
}

@JsonSerializable()
class CreatePackResponseDto extends Equatable {
  const CreatePackResponseDto({
    required this.id,
    required this.uploadPrefix,
  });

  factory CreatePackResponseDto.fromJson(Map<String, dynamic> json) =>
      _$CreatePackResponseDtoFromJson(json);

  final String id;
  @JsonKey(name: "upload_prefix")
  final String uploadPrefix;

  Map<String, dynamic> toJson() => _$CreatePackResponseDtoToJson(this);

  @override
  List<Object?> get props => <Object?>[id, uploadPrefix];
}

@JsonSerializable()
class RejectionStatDto extends Equatable {
  const RejectionStatDto({
    required this.reasonCode,
    required this.rejectionCount,
    required this.lastRejectedAt,
  });

  factory RejectionStatDto.fromJson(Map<String, dynamic> json) =>
      _$RejectionStatDtoFromJson(json);

  @JsonKey(name: "reason_code", defaultValue: "other")
  final String reasonCode;
  @JsonKey(name: "rejection_count", defaultValue: 0)
  final int rejectionCount;
  @JsonKey(name: "last_rejected_at", defaultValue: "")
  final String lastRejectedAt;

  Map<String, dynamic> toJson() => _$RejectionStatDtoToJson(this);

  RejectionStat toEntity() {
    return RejectionStat(
      reasonCode: reasonCode,
      rejectionCount: rejectionCount,
      lastRejectedAt: lastRejectedAt,
    );
  }

  @override
  List<Object?> get props =>
      <Object?>[reasonCode, rejectionCount, lastRejectedAt];
}

@JsonSerializable()
class RejectionAnalyticsResponseDto extends Equatable {
  const RejectionAnalyticsResponseDto({required this.analytics});

  factory RejectionAnalyticsResponseDto.fromJson(Map<String, dynamic> json) =>
      _$RejectionAnalyticsResponseDtoFromJson(json);

  final List<RejectionStatDto> analytics;

  Map<String, dynamic> toJson() => _$RejectionAnalyticsResponseDtoToJson(this);

  @override
  List<Object?> get props => <Object?>[analytics];
}
