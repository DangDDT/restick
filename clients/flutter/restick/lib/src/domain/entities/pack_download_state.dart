import "package:equatable/equatable.dart";

enum PackDownloadStatus {
  remote,
  downloading,
  downloaded,
  failed,
  revoked,
}

class PackDownloadState extends Equatable {
  const PackDownloadState({
    required this.packId,
    required this.status,
    this.progress = 0,
    this.message = "",
  });

  final String packId;
  final PackDownloadStatus status;
  final double progress;
  final String message;

  bool get isDownloaded => status == PackDownloadStatus.downloaded;

  PackDownloadState copyWith({
    PackDownloadStatus? status,
    double? progress,
    String? message,
  }) {
    return PackDownloadState(
      packId: packId,
      status: status ?? this.status,
      progress: progress ?? this.progress,
      message: message ?? this.message,
    );
  }

  @override
  List<Object?> get props => <Object?>[packId, status, progress, message];
}
