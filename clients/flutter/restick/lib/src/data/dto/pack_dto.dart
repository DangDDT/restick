import "package:restick/src/data/mappers/json_codec.dart";
import "package:restick/src/data/mappers/url_rewriter.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/tombstone.dart";

class PackSummaryDto {
  const PackSummaryDto({
    required this.id,
    required this.slug,
    required this.name,
    required this.author,
    required this.tags,
    required this.stickerCount,
    required this.version,
    required this.thumbnailUrl,
    required this.checksumSha256,
  });

  factory PackSummaryDto.fromJson(Map<String, dynamic> json) {
    return PackSummaryDto(
      id: json["id"]! as String,
      slug: json["slug"]! as String,
      name: json["name"]! as String,
      author: json["author"]! as String,
      tags: asStringList(json["tags"]),
      stickerCount: (json["sticker_count"]! as num).toInt(),
      version: (json["version"]! as num).toInt(),
      thumbnailUrl: (json["thumbnail_url"] as String?) ?? "",
      checksumSha256: (json["checksum_sha256"] as String?) ?? "",
    );
  }

  final String id;
  final String slug;
  final String name;
  final String author;
  final List<String> tags;
  final int stickerCount;
  final int version;
  final String thumbnailUrl;
  final String checksumSha256;

  StickerPack toEntity(Uri apiBase) {
    return StickerPack(
      id: id,
      slug: slug,
      name: name,
      author: author,
      tags: tags,
      stickerCount: stickerCount,
      version: version,
      thumbnailUrl: rewritePublicUrl(thumbnailUrl, apiBase),
      checksumSha256: checksumSha256,
    );
  }
}

class PackListResponseDto {
  const PackListResponseDto({required this.items});

  factory PackListResponseDto.fromJson(Map<String, dynamic> json) {
    return PackListResponseDto(
      items: asJsonList(json["items"])
          .map(
            (Object? item) =>
                PackSummaryDto.fromJson(asJsonMap(item).cast<String, dynamic>()),
          )
          .toList(),
    );
  }

  final List<PackSummaryDto> items;
}

class BundleMetaDto {
  const BundleMetaDto({
    required this.url,
    required this.checksumSha256,
    required this.version,
  });

  factory BundleMetaDto.fromJson(Map<String, dynamic> json) {
    return BundleMetaDto(
      url: json["url"]! as String,
      checksumSha256: json["checksum_sha256"]! as String,
      version: (json["version"]! as num).toInt(),
    );
  }

  final String url;
  final String checksumSha256;
  final int version;

  String rewrittenUrl(Uri apiBase) => rewritePublicUrl(url, apiBase);
}

class TombstoneDto {
  const TombstoneDto({
    required this.packId,
    required this.revokedAt,
    required this.reasonCode,
    required this.checksumSha256,
  });

  factory TombstoneDto.fromJson(Map<String, dynamic> json) {
    return TombstoneDto(
      packId: json["pack_id"]! as String,
      revokedAt: json["revoked_at"]! as String,
      reasonCode: json["reason_code"]! as String,
      checksumSha256: (json["checksum_sha256"] as String?) ?? "",
    );
  }

  final String packId;
  final String revokedAt;
  final String reasonCode;
  final String checksumSha256;

  Tombstone toEntity() {
    return Tombstone(
      packId: packId,
      revokedAt: revokedAt,
      reasonCode: reasonCode,
      checksumSha256: checksumSha256,
    );
  }
}

class TombstoneListResponseDto {
  const TombstoneListResponseDto({required this.items});

  factory TombstoneListResponseDto.fromJson(Map<String, dynamic> json) {
    return TombstoneListResponseDto(
      items: asJsonList(json["items"])
          .map(
            (Object? item) =>
                TombstoneDto.fromJson(asJsonMap(item).cast<String, dynamic>()),
          )
          .toList(),
    );
  }

  final List<TombstoneDto> items;
}
