import "dart:convert";
import "dart:io";

import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";
import "package:path/path.dart" as p;
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/presentation/widgets/sticker_grid_tile.dart";

// Real 2x2 lossless WebP (Pillow, method=6) and a real two-frame animated
// WebP, so the tile decodes actual bytes instead of a fake codec.
const String _staticWebp =
    "UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAdQ5KJUq/+BiOh/AAA=";
const String _animatedWebp =
    "UklGRoQAAABXRUJQVlA4WAoAAAACAAAAAQAAAQAAQU5JTQYAAAAAAAAAAABBTk1GKAAAAAAAAAAAAAEAAAEAAGQAAAJWUDhMDwAAAC8BQAAABxD9j/4HIqL/AQBBTk1GKAAAAAAAAAAAAAEAAAEAAGQAAABWUDhMDwAAAC8BQAAABxDR//4HIqL/AQA=";

void main() {
  late Directory dir;
  late StickerAsset animated;
  late StickerAsset plain;

  setUp(() async {
    dir = await Directory.systemTemp.createTemp("restick_tile_");
    final File staticFile = File(p.join(dir.path, "000.webp"))
      ..writeAsBytesSync(base64Decode(_staticWebp));
    final File thumb = File(p.join(dir.path, "thumbs", "thumb_000.webp"))
      ..createSync(recursive: true)
      ..writeAsBytesSync(base64Decode(_staticWebp));
    final File animatedFile = File(p.join(dir.path, "animated", "000.webp"))
      ..createSync(recursive: true)
      ..writeAsBytesSync(base64Decode(_animatedWebp));
    animated = StickerAsset(
      packId: "pack",
      file: staticFile,
      thumb: thumb,
      emojiHint: "",
      width: 512,
      height: 512,
      animatedFile: animatedFile,
    );
    plain = StickerAsset(
      packId: "pack",
      file: staticFile,
      thumb: thumb,
      emojiHint: "",
      width: 512,
      height: 512,
    );
  });

  tearDown(() => dir.delete(recursive: true));

  Future<String> shownPath(
    WidgetTester tester, {
    required bool disableAnimations,
    StickerAsset? sticker,
  }) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: MediaQuery(
            data: MediaQueryData(disableAnimations: disableAnimations),
            child: StickerGridTile(
              asset: sticker ?? animated,
              onSelected: (StickerAsset _) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final Image image = tester.widget<Image>(find.byType(Image));
    final ImageProvider<Object> provider = image.image;
    final ImageProvider<Object> fileProvider =
        provider is ResizeImage ? provider.imageProvider : provider;
    return (fileProvider as FileImage).file.path;
  }

  testWidgets("tile shows thumb when disableAnimations is true",
      (WidgetTester tester) async {
    expect(
      await shownPath(tester, disableAnimations: true),
      animated.thumb.path,
    );
  });

  testWidgets("tile uses animated file when enabled and present",
      (WidgetTester tester) async {
    expect(
      await shownPath(tester, disableAnimations: false),
      animated.animatedFile!.path,
    );
  });

  testWidgets("tile keeps the thumb for static assets",
      (WidgetTester tester) async {
    expect(
      await shownPath(tester, disableAnimations: false, sticker: plain),
      plain.thumb.path,
    );
  });
}
