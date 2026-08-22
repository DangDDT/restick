import "package:equatable/equatable.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";

class StickerPack extends Equatable {
  const StickerPack({
    required this.id,
    required this.slug,
    required this.name,
    required this.author,
    required this.tags,
    required this.stickerCount,
    required this.version,
    required this.thumbnailUrl,
    required this.checksumSha256,
    this.downloadStatus = PackDownloadStatus.remote,
    this.downloadProgress = 0,
    this.localDir = "",
  });

  final String id;
  final String slug;
  final String name;
  final String author;
  final List<String> tags;
  final int stickerCount;
  final int version;
  final String thumbnailUrl;
  final String checksumSha256;
  final PackDownloadStatus downloadStatus;
  final double downloadProgress;
  final String localDir;

  bool get isDownloaded => downloadStatus == PackDownloadStatus.downloaded;

  StickerPack copyWith({
    PackDownloadStatus? downloadStatus,
    double? downloadProgress,
    String? localDir,
    int? version,
    String? checksumSha256,
    String? thumbnailUrl,
    int? stickerCount,
    List<String>? tags,
    String? name,
    String? author,
  }) {
    return StickerPack(
      id: id,
      slug: slug,
      name: name ?? this.name,
      author: author ?? this.author,
      tags: tags ?? this.tags,
      stickerCount: stickerCount ?? this.stickerCount,
      version: version ?? this.version,
      thumbnailUrl: thumbnailUrl ?? this.thumbnailUrl,
      checksumSha256: checksumSha256 ?? this.checksumSha256,
      downloadStatus: downloadStatus ?? this.downloadStatus,
      downloadProgress: downloadProgress ?? this.downloadProgress,
      localDir: localDir ?? this.localDir,
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
        thumbnailUrl,
        checksumSha256,
        downloadStatus,
        downloadProgress,
        localDir,
      ];
}
