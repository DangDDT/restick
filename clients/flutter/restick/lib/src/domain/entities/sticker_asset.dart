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
    this.animatedFile,
    this.keywords = const <String>[],
  });

  final String packId;
  final File file;
  final File thumb;
  final String emojiHint;
  final int width;
  final int height;

  /// Animated WebP variant, null for a static-only sticker or when the local
  /// file is gone.
  final File? animatedFile;

  /// Search keywords from the pack manifest; empty on older packs.
  final List<String> keywords;

  bool get isAnimated => animatedFile != null;

  File get preview => thumb.existsSync() ? thumb : file;

  @override
  List<Object?> get props => <Object?>[
    packId,
    file.path,
    thumb.path,
    emojiHint,
    width,
    height,
    animatedFile?.path,
    keywords,
  ];
}
