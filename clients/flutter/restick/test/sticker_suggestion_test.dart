import "dart:io";

import "package:flutter_test/flutter_test.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/sticker_suggestion.dart";

StickerAsset _sticker(
  String name,
  String emoji, [
  List<String> keywords = const <String>[],
]) {
  return StickerAsset(
    packId: "p",
    file: File(name),
    thumb: File("thumb_$name"),
    emojiHint: emoji,
    width: 512,
    height: 512,
    keywords: keywords,
  );
}

List<String> _names(List<StickerAsset> list) =>
    list.map((StickerAsset s) => s.file.path).toList();

void main() {
  final StickerAsset thanks = _sticker("thanks", "🙏", <String>[
    "thanks",
    "cam on",
    "bowing",
  ]);
  final StickerAsset haha = _sticker("haha", "😂", <String>["haha", "lol"]);
  final StickerAsset bare = _sticker("bare", "");
  final List<StickerAsset> all = <StickerAsset>[thanks, haha, bare];

  test("matches a keyword ignoring case, punctuation and diacritics", () {
    expect(_names(suggestStickersFor(all, "Cảm ơn!")), <String>["thanks"]);
    expect(_names(suggestStickersFor(all, "CAM ON bạn nhé.")), <String>[
      "thanks",
    ]);
    expect(_names(suggestStickersFor(all, "Thanks!!")), <String>["thanks"]);
  });

  test("keywords match whole words only", () {
    expect(suggestStickersFor(all, "thankful"), isEmpty);
    expect(suggestStickersFor(all, "lollipop"), isEmpty);
  });

  test("matches the emoji hint and ranks it above keywords", () {
    expect(_names(suggestStickersFor(all, "lol 🙏")), <String>[
      "thanks",
      "haha",
    ]);
  });

  test("empty text, no metadata and limit", () {
    expect(suggestStickersFor(all, "  "), isEmpty);
    expect(_names(suggestStickersFor(all, "lol thanks", limit: 1)), <String>[
      "thanks",
    ]);
  });
}
