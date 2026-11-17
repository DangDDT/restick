import "package:equatable/equatable.dart";

/// One sticker shown in a marketplace pack card or banner.
class StickerPreview extends Equatable {
  const StickerPreview({required this.thumbUrl, required this.fileUrl});

  final String thumbUrl;
  final String fileUrl;

  @override
  List<Object?> get props => <Object?>[thumbUrl, fileUrl];
}
