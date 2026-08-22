import "dart:io";

import "package:equatable/equatable.dart";

class StickerAsset extends Equatable {
  const StickerAsset({
    required this.packId,
    required this.file,
    required this.thumb,
    required this.emojiHint,
    required this.width,
    required this.height,
  });

  final String packId;
  final File file;
  final File thumb;
  final String emojiHint;
  final int width;
  final int height;

  File get preview => thumb.existsSync() ? thumb : file;

  @override
  List<Object?> get props => <Object?>[
        packId,
        file.path,
        thumb.path,
        emojiHint,
        width,
        height,
      ];
}
